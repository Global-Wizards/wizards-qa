package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Global-Wizards/wizards-qa/web/backend/auth"
	"github.com/Global-Wizards/wizards-qa/web/backend/store"
	"github.com/Global-Wizards/wizards-qa/web/backend/ws"
)

type AnalysisRequest struct {
	GameURL   string `json:"gameUrl"`
	ProjectID string `json:"projectId"`
	AgentMode bool   `json:"agentMode"`
}

type AnalysisProgress struct {
	Step    string      `json:"step"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (s *Server) handleAnalyzeGame(w http.ResponseWriter, r *http.Request) {
	var req AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.GameURL == "" {
		respondError(w, http.StatusBadRequest, "gameUrl is required")
		return
	}

	parsed, err := url.ParseRequestURI(req.GameURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		respondError(w, http.StatusBadRequest, "Invalid URL: must start with http:// or https://")
		return
	}

	analysisID := fmt.Sprintf("analysis-%d", time.Now().UnixNano())

	// Get createdBy from auth context
	var createdBy string
	if claims := auth.UserFromContext(r.Context()); claims != nil {
		createdBy = claims.UserID
	}

	projectID := req.ProjectID

	agentMode := req.AgentMode

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in analysis %s: %v", analysisID, r)
				s.wsHub.Broadcast(ws.Message{
					Type: "analysis_failed",
					Data: map[string]string{
						"analysisId": analysisID,
						"error":      fmt.Sprintf("panic: %v", r),
					},
				})
			}
		}()
		s.executeAnalysis(analysisID, req.GameURL, createdBy, projectID, agentMode)
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"analysisId": analysisID,
		"status":     "started",
		"message":    "Analysis started",
	})
}

func (s *Server) executeAnalysis(analysisID, gameURL, createdBy, projectID string, agentMode bool) {
	// Acquire concurrency slot
	select {
	case s.analysisSem <- struct{}{}:
		defer func() { <-s.analysisSem }()
	case <-s.serverCtx.Done():
		s.broadcastAnalysisError(analysisID, "Server shutting down")
		return
	}

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_progress",
		Data: AnalysisProgress{
			Step:    "scouting",
			Message: "Scouting page...",
			Data:    map[string]string{"analysisId": analysisID, "gameUrl": gameURL},
		},
	})

	// Save "running" record immediately so the job is persisted
	runningRecord := store.AnalysisRecord{
		ID:        analysisID,
		GameURL:   gameURL,
		Status:    store.StatusRunning,
		Step:      "scouting",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
		CreatedBy: createdBy,
		ProjectID: projectID,
	}
	if err := s.store.SaveAnalysis(runningRecord); err != nil {
		log.Printf("Warning: failed to save running analysis record for %s: %v", analysisID, err)
	}

	cliPath := envOrDefault("WIZARDS_QA_CLI_PATH", "wizards-qa")

	tmpDir, err := os.MkdirTemp("", "wizards-qa-analysis-*")
	if err != nil {
		s.broadcastAnalysisError(analysisID, fmt.Sprintf("Failed to create temp dir: %v", err))
		return
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := context.WithTimeout(s.serverCtx, 5*time.Minute)
	defer cancel()

	args := []string{"scout", "--game", gameURL, "--json", "--save-flows", "--output", tmpDir, "--headless", "--timeout", "60"}
	if agentMode {
		args = append(args, "--agent")
	}
	log.Printf("Analysis %s: executing %s %s", analysisID, cliPath, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.broadcastAnalysisError(analysisID, fmt.Sprintf("stdout pipe: %v", err))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.broadcastAnalysisError(analysisID, fmt.Sprintf("stderr pipe: %v", err))
		return
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.broadcastAnalysisError(analysisID, fmt.Sprintf("stdin pipe: %v", err))
		return
	}

	if err := cmd.Start(); err != nil {
		s.broadcastAnalysisError(analysisID, fmt.Sprintf("Failed to start CLI: %v", err))
		return
	}

	// Register active analysis for user→agent messaging
	s.activeAnalysesMu.Lock()
	s.activeAnalyses[analysisID] = &activeAnalysis{stdin: stdin, tmpDir: tmpDir}
	s.activeAnalysesMu.Unlock()
	defer func() {
		s.activeAnalysesMu.Lock()
		delete(s.activeAnalyses, analysisID)
		s.activeAnalysesMu.Unlock()
		stdin.Close()
	}()

	// Stream stderr for PROGRESS: lines and collect non-progress lines for error reporting
	var stderrLines []string
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		stderrScanner := bufio.NewScanner(stderr)
		stderrScanner.Buffer(make([]byte, 256*1024), 256*1024)
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			if strings.HasPrefix(line, "PROGRESS:") {
				rest := line[len("PROGRESS:"):]
				parts := strings.SplitN(rest, ":", 2)
				step := strings.TrimSpace(parts[0])
				message := ""
				if len(parts) > 1 {
					message = strings.TrimSpace(parts[1])
				}

				// Handle rich agent events with dedicated WS message types
				switch step {
				case "agent_step_detail":
					var detailData map[string]interface{}
					if err := json.Unmarshal([]byte(message), &detailData); err == nil {
						detailData["analysisId"] = analysisID
						s.wsHub.Broadcast(ws.Message{
							Type: "agent_step_detail",
							Data: detailData,
						})
					}
				case "agent_reasoning":
					s.wsHub.Broadcast(ws.Message{
						Type: "agent_reasoning",
						Data: map[string]string{
							"analysisId": analysisID,
							"text":       message,
						},
					})
				case "agent_screenshot":
					// Read screenshot file from tmpDir and broadcast as base64
					filename := filepath.Base(message) // strip directory components
					if filename == "." || filename == "/" || strings.ContainsAny(filename, `/\`) {
						break
					}
					s.activeAnalysesMu.Lock()
					aa := s.activeAnalyses[analysisID]
					s.activeAnalysesMu.Unlock()
					if aa != nil {
						screenshotPath := filepath.Join(aa.tmpDir, "agent-screenshots", filename)
						if imgData, readErr := os.ReadFile(screenshotPath); readErr == nil {
							s.wsHub.Broadcast(ws.Message{
								Type: "agent_screenshot",
								Data: map[string]string{
									"analysisId": analysisID,
									"imageData":  base64.StdEncoding.EncodeToString(imgData),
									"filename":   filename,
								},
							})
						}
					}
				case "user_hint":
					s.wsHub.Broadcast(ws.Message{
						Type: "agent_user_hint",
						Data: map[string]string{
							"analysisId": analysisID,
							"message":    message,
						},
					})
				}

				// Always broadcast as analysis_progress too (for backward compat)
				s.wsHub.Broadcast(ws.Message{
					Type: "analysis_progress",
					Data: AnalysisProgress{
						Step:    step,
						Message: message,
						Data:    map[string]string{"analysisId": analysisID},
					},
				})
				go func(id, st string) {
					if err := s.store.UpdateAnalysisStatus(id, store.StatusRunning, st); err != nil {
						log.Printf("Warning: failed to update analysis %s step to %s: %v", id, st, err)
					}
				}(analysisID, step)
			} else {
				stderrLines = append(stderrLines, line)
				log.Printf("Analysis %s stderr: %s", analysisID, line)
			}
		}
	}()

	// Collect all stdout
	var outputBuf bytes.Buffer
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		outputBuf.WriteString(line)
		outputBuf.WriteString("\n")
	}

	if scanErr := scanner.Err(); scanErr != nil {
		log.Printf("Warning: scanner error reading analysis output for %s: %v", analysisID, scanErr)
	}

	<-stderrDone

	err = cmd.Wait()
	if err != nil {
		errMsg := err.Error()
		if len(stderrLines) > 0 {
			errMsg = strings.Join(stderrLines, "\n")
		}
		s.broadcastAnalysisError(analysisID, errMsg)
		return
	}

	// Parse the JSON output
	rawOutput := strings.TrimSpace(outputBuf.String())
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(rawOutput), &result); err != nil {
		start := strings.Index(rawOutput, "{")
		end := strings.LastIndex(rawOutput, "}")
		if start >= 0 && end > start {
			rawOutput = rawOutput[start : end+1]
			if err2 := json.Unmarshal([]byte(rawOutput), &result); err2 != nil {
				s.broadcastAnalysisError(analysisID, fmt.Sprintf("Failed to parse CLI output: %v (fallback: %v)", err, err2))
				return
			}
		} else {
			s.broadcastAnalysisError(analysisID, fmt.Sprintf("Failed to parse CLI output: %v", err))
			return
		}
	}

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_progress",
		Data: AnalysisProgress{
			Step:    "saving",
			Message: "Saving generated flows...",
			Data:    map[string]string{"analysisId": analysisID},
		},
	})

	if err := s.store.SaveGeneratedFlows(analysisID, tmpDir); err != nil {
		log.Printf("Warning: failed to save generated flows for %s: %v", analysisID, err)
	}

	flowCount := 0
	if flows, ok := result["flows"]; ok {
		if flowSlice, ok := flows.([]interface{}); ok {
			flowCount = len(flowSlice)
		}
	}

	gameName := ""
	framework := ""
	if pm, ok := result["pageMeta"].(map[string]interface{}); ok {
		if t, ok := pm["title"].(string); ok {
			gameName = t
		}
		if f, ok := pm["framework"].(string); ok {
			framework = f
		}
	}
	if analysis, ok := result["analysis"].(map[string]interface{}); ok {
		if gi, ok := analysis["gameInfo"].(map[string]interface{}); ok {
			if n, ok := gi["name"].(string); ok && n != "" {
				gameName = n
			}
		}
	}

	if err := s.store.UpdateAnalysisResult(analysisID, store.StatusCompleted, result, gameName, framework, flowCount); err != nil {
		log.Printf("Warning: failed to update analysis record for %s: %v", analysisID, err)
	}

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_completed",
		Data: map[string]interface{}{
			"analysisId": analysisID,
			"result":     result,
		},
	})
}

func (s *Server) handleSendAgentMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		respondError(w, http.StatusBadRequest, "message is required")
		return
	}

	// Limit message length
	if len(req.Message) > 500 {
		req.Message = req.Message[:500]
	}

	s.activeAnalysesMu.Lock()
	aa, ok := s.activeAnalyses[id]
	if !ok {
		s.activeAnalysesMu.Unlock()
		respondError(w, http.StatusGone, "Analysis is not running")
		return
	}

	// Rate limit: 5s cooldown per analysis
	if time.Since(aa.lastHintAt) < 5*time.Second {
		s.activeAnalysesMu.Unlock()
		respondError(w, http.StatusTooManyRequests, "Please wait before sending another hint")
		return
	}

	// Write JSON line to stdin pipe while still holding the lock
	// to prevent a write-to-closed-pipe race if the analysis ends concurrently.
	hintLine := map[string]string{"type": "user_hint", "message": req.Message}
	hintJSON, _ := json.Marshal(hintLine)
	hintJSON = append(hintJSON, '\n')
	_, writeErr := aa.stdin.Write(hintJSON)
	if writeErr == nil {
		aa.lastHintAt = time.Now()
	}
	s.activeAnalysesMu.Unlock()

	if writeErr != nil {
		respondError(w, http.StatusGone, "Analysis has ended")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) broadcastAnalysisError(analysisID, errMsg string) {
	log.Printf("Analysis %s failed: %s", analysisID, errMsg)

	if err := s.store.UpdateAnalysisStatus(analysisID, store.StatusFailed, ""); err != nil {
		log.Printf("Warning: failed to mark analysis %s as failed: %v", analysisID, err)
	}

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_failed",
		Data: map[string]string{
			"analysisId": analysisID,
			"error":      errMsg,
		},
	})
}
