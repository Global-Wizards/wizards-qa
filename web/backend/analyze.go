package main

import (
	"bufio"
	"bytes"
	"context"
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

type AnalysisModules struct {
	UIUX       *bool `json:"uiux,omitempty"`
	Wording    *bool `json:"wording,omitempty"`
	GameDesign *bool `json:"gameDesign,omitempty"`
	TestFlows  *bool `json:"testFlows,omitempty"`
	RunTests   *bool `json:"runTests,omitempty"`
}

type AnalysisRequest struct {
	GameURL         string          `json:"gameUrl"`
	ProjectID       string          `json:"projectId"`
	AgentMode       bool            `json:"agentMode"`
	Modules         AnalysisModules `json:"modules"`
	Model           string          `json:"model,omitempty"`
	MaxTokens       int             `json:"maxTokens,omitempty"`
	AgentSteps      int             `json:"agentSteps,omitempty"`
	Temperature     *float64        `json:"temperature,omitempty"`
	Adaptive        bool            `json:"adaptive,omitempty"`
	MaxTotalSteps   int             `json:"maxTotalSteps,omitempty"`
	AdaptiveTimeout bool            `json:"adaptiveTimeout,omitempty"`
	MaxTotalTimeout int             `json:"maxTotalTimeout,omitempty"` // minutes
	Viewport        string          `json:"viewport,omitempty"`       // viewport preset name
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

	// Validate profile param bounds
	if req.MaxTokens != 0 && (req.MaxTokens < 256 || req.MaxTokens > 32768) {
		respondError(w, http.StatusBadRequest, "maxTokens must be between 256 and 32768")
		return
	}
	if req.AgentSteps != 0 && (req.AgentSteps < 1 || req.AgentSteps > 100) {
		respondError(w, http.StatusBadRequest, "agentSteps must be between 1 and 100")
		return
	}
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 1) {
		respondError(w, http.StatusBadRequest, "temperature must be between 0.0 and 1.0")
		return
	}
	if req.MaxTotalSteps != 0 && (req.MaxTotalSteps < 1 || req.MaxTotalSteps > 100) {
		respondError(w, http.StatusBadRequest, "maxTotalSteps must be between 1 and 100")
		return
	}
	if req.MaxTotalSteps > 0 && req.AgentSteps > 0 && req.MaxTotalSteps < req.AgentSteps {
		respondError(w, http.StatusBadRequest, "maxTotalSteps must be >= agentSteps")
		return
	}
	if req.MaxTotalTimeout != 0 && (req.MaxTotalTimeout < 1 || req.MaxTotalTimeout > 60) {
		respondError(w, http.StatusBadRequest, "maxTotalTimeout must be between 1 and 60 minutes")
		return
	}

	analysisID := newID("analysis")

	// Get createdBy from auth context
	var createdBy string
	if claims := auth.UserFromContext(r.Context()); claims != nil {
		createdBy = claims.UserID
	}

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
		s.executeAnalysis(analysisID, createdBy, req)
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"analysisId": analysisID,
		"status":     "started",
		"message":    "Analysis started",
	})
}

// BatchDeviceConfig specifies a device for batch analysis.
type BatchDeviceConfig struct {
	Category string `json:"category"` // "desktop", "ios", "android"
	Viewport string `json:"viewport"` // viewport preset name
}

// BatchAnalysisRequest runs analyses for multiple device viewports simultaneously.
type BatchAnalysisRequest struct {
	GameURL         string            `json:"gameUrl"`
	ProjectID       string            `json:"projectId,omitempty"`
	AgentMode       bool              `json:"agentMode"`
	Modules         AnalysisModules   `json:"modules"`
	Devices         []BatchDeviceConfig `json:"devices"`
	Model           string            `json:"model,omitempty"`
	MaxTokens       int               `json:"maxTokens,omitempty"`
	AgentSteps      int               `json:"agentSteps,omitempty"`
	Temperature     *float64          `json:"temperature,omitempty"`
	Adaptive        bool              `json:"adaptive,omitempty"`
	MaxTotalSteps   int               `json:"maxTotalSteps,omitempty"`
	AdaptiveTimeout bool              `json:"adaptiveTimeout,omitempty"`
	MaxTotalTimeout int               `json:"maxTotalTimeout,omitempty"`
}

func (s *Server) handleBatchAnalyze(w http.ResponseWriter, r *http.Request) {
	var req BatchAnalysisRequest
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

	if len(req.Devices) == 0 {
		respondError(w, http.StatusBadRequest, "At least one device is required")
		return
	}
	if len(req.Devices) > 5 {
		respondError(w, http.StatusBadRequest, "Maximum 5 devices per batch")
		return
	}

	var createdBy string
	if claims := auth.UserFromContext(r.Context()); claims != nil {
		createdBy = claims.UserID
	}

	analysisID := newID("analysis")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in batch analysis %s: %v", analysisID, r)
				s.wsHub.Broadcast(ws.Message{
					Type: "analysis_failed",
					Data: map[string]string{
						"analysisId": analysisID,
						"error":      fmt.Sprintf("panic: %v", r),
					},
				})
			}
		}()
		s.executeBatchAnalysis(analysisID, createdBy, req)
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"analysisId": analysisID,
		"status":     "started",
		"devices":    req.Devices,
	})
}

// deviceResult holds the outcome of a single device's analysis within a batch.
type deviceResult struct {
	Device    string                 `json:"device"`
	Viewport  string                 `json:"viewport"`
	FlowCount int                    `json:"flowCount"`
	Status    string                 `json:"status"` // "completed" or "failed"
	Error     string                 `json:"error,omitempty"`
	Flows     []interface{}          `json:"-"`
	PageMeta  map[string]interface{} `json:"-"`
	Analysis  map[string]interface{} `json:"-"`
	Mode      string                 `json:"-"`
}

func (s *Server) executeBatchAnalysis(analysisID, createdBy string, req BatchAnalysisRequest) {
	gameURL := req.GameURL
	agentMode := req.AgentMode

	// Serialize modules to JSON for persistence
	modulesJSON := ""
	{
		m := map[string]bool{
			"uiux":       req.Modules.UIUX == nil || *req.Modules.UIUX,
			"wording":    req.Modules.Wording == nil || *req.Modules.Wording,
			"gameDesign": req.Modules.GameDesign == nil || *req.Modules.GameDesign,
			"testFlows":  req.Modules.TestFlows == nil || *req.Modules.TestFlows,
			"runTests":   req.Modules.RunTests != nil && *req.Modules.RunTests,
		}
		if b, err := json.Marshal(m); err == nil {
			modulesJSON = string(b)
		}
	}

	// Serialize profile params for persistence
	profileJSON := ""
	{
		p := map[string]interface{}{}
		if req.Model != "" {
			p["model"] = req.Model
		}
		if req.MaxTokens > 0 {
			p["maxTokens"] = req.MaxTokens
		}
		if req.Temperature != nil {
			p["temperature"] = *req.Temperature
		}
		if req.AgentSteps > 0 {
			p["agentSteps"] = req.AgentSteps
		}
		if req.Adaptive {
			p["adaptive"] = true
		}
		if req.MaxTotalSteps > 0 {
			p["maxTotalSteps"] = req.MaxTotalSteps
		}
		if req.AdaptiveTimeout {
			p["adaptiveTimeout"] = true
		}
		if req.MaxTotalTimeout > 0 {
			p["maxTotalTimeout"] = req.MaxTotalTimeout
		}
		// Store device configs in profile for reference
		devicesJSON, _ := json.Marshal(req.Devices)
		p["devices"] = string(devicesJSON)
		if len(p) > 0 {
			if b, err := json.Marshal(p); err == nil {
				profileJSON = string(b)
			}
		}
	}

	// Save ONE running record before acquiring semaphore
	runningRecord := store.AnalysisRecord{
		ID:        analysisID,
		GameURL:   gameURL,
		Status:    store.StatusRunning,
		Step:      "queued",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
		CreatedBy: createdBy,
		ProjectID: req.ProjectID,
		Modules:   modulesJSON,
		AgentMode: agentMode,
		Profile:   profileJSON,
	}
	if err := s.store.SaveAnalysis(runningRecord); err != nil {
		log.Printf("Warning: failed to save running analysis record for %s: %v", analysisID, err)
	}

	// Acquire semaphore once for the entire batch
	select {
	case s.analysisSem <- struct{}{}:
	default:
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "queued",
				Message: "Another analysis is running. Waiting in queue...",
				Data:    map[string]string{"analysisId": analysisID},
			},
		})
		log.Printf("Batch analysis %s: queued (semaphore busy)", analysisID)
		queueTimeout := time.After(5 * time.Minute)
		select {
		case s.analysisSem <- struct{}{}:
		case <-queueTimeout:
			s.store.UpdateAnalysisStatus(analysisID, store.StatusFailed, "queue_timeout")
			s.broadcastAnalysisError(analysisID, "Timed out waiting in queue. Another analysis may be stuck — please try again.")
			return
		case <-s.serverCtx.Done():
			s.store.UpdateAnalysisStatus(analysisID, store.StatusFailed, "shutdown")
			s.broadcastAnalysisError(analysisID, "Server shutting down")
			return
		}
	}
	defer func() { <-s.analysisSem }()

	// Update step now that we have the semaphore
	if err := s.store.UpdateAnalysisStatus(analysisID, store.StatusRunning, "scouting"); err != nil {
		log.Printf("Warning: failed to update analysis %s step to scouting: %v", analysisID, err)
	}

	// Calculate total timeout: per-device timeout × device count, clamped to 60 min
	perDeviceTimeout := 5 * time.Minute
	if agentMode {
		steps := req.AgentSteps
		if steps <= 0 {
			steps = 20
		}
		if req.Adaptive && req.MaxTotalSteps > steps {
			steps = req.MaxTotalSteps
		}
		explorationBudget := time.Duration(steps) * 75 * time.Second
		perDeviceTimeout = explorationBudget + 10*time.Minute
		if req.AdaptiveTimeout && req.MaxTotalTimeout > 0 {
			timeoutFromMinutes := time.Duration(req.MaxTotalTimeout)*time.Minute + 8*time.Minute
			if timeoutFromMinutes > perDeviceTimeout {
				perDeviceTimeout = timeoutFromMinutes
			}
		}
		if perDeviceTimeout < 10*time.Minute {
			perDeviceTimeout = 10 * time.Minute
		}
		maxClamp := 45 * time.Minute
		if req.Adaptive || req.AdaptiveTimeout {
			maxClamp = 60 * time.Minute
		}
		if perDeviceTimeout > maxClamp {
			perDeviceTimeout = maxClamp
		}
	}
	totalTimeout := perDeviceTimeout * time.Duration(len(req.Devices))
	if totalTimeout > 60*time.Minute {
		totalTimeout = 60 * time.Minute
	}
	ctx, cancel := context.WithTimeout(s.serverCtx, totalTimeout)
	defer cancel()

	cliPath := envOrDefault("WIZARDS_QA_CLI_PATH", "wizards-qa")

	var deviceResults []deviceResult
	var allFlows []interface{}
	var allAgentSteps []interface{}
	var primaryPageMeta map[string]interface{}
	var primaryAnalysis map[string]interface{}
	var primaryMode string
	var gameName, framework string
	totalFlowCount := 0
	agentStepOffset := 0

	for i, device := range req.Devices {
		deviceNum := i + 1
		deviceTotal := len(req.Devices)
		deviceLabel := fmt.Sprintf("[%s %d/%d]", device.Category, deviceNum, deviceTotal)

		// Check if context is already cancelled
		if ctx.Err() != nil {
			deviceResults = append(deviceResults, deviceResult{
				Device:   device.Category,
				Viewport: device.Viewport,
				Status:   "failed",
				Error:    "Batch timed out before this device could run",
			})
			continue
		}

		// Broadcast device transition (for devices after the first)
		if i > 0 {
			s.wsHub.Broadcast(ws.Message{
				Type: "analysis_progress",
				Data: AnalysisProgress{
					Step:    "device_transition",
					Message: fmt.Sprintf("%s Starting analysis...", deviceLabel),
					Data: map[string]string{
						"analysisId":  analysisID,
						"device":      device.Category,
						"deviceIndex": fmt.Sprintf("%d", deviceNum),
						"deviceTotal": fmt.Sprintf("%d", deviceTotal),
					},
				},
			})

			// Broadcast an agent_reasoning separator between devices
			s.wsHub.Broadcast(ws.Message{
				Type: "agent_reasoning",
				Data: map[string]string{
					"analysisId": analysisID,
					"text":       fmt.Sprintf("--- Starting %s analysis (device %d of %d) ---", device.Category, deviceNum, deviceTotal),
				},
			})
		}

		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "scouting",
				Message: fmt.Sprintf("%s Scouting page...", deviceLabel),
				Data: map[string]string{
					"analysisId":  analysisID,
					"device":      device.Category,
					"deviceIndex": fmt.Sprintf("%d", deviceNum),
					"deviceTotal": fmt.Sprintf("%d", deviceTotal),
					"gameUrl":     gameURL,
				},
			},
		})

		if err := s.store.UpdateAnalysisStatus(analysisID, store.StatusRunning, "scouting"); err != nil {
			log.Printf("Warning: failed to update analysis %s step to scouting: %v", analysisID, err)
		}

		// Create per-device tmpDir
		tmpDir, err := os.MkdirTemp("", fmt.Sprintf("wizards-qa-batch-%s-*", device.Category))
		if err != nil {
			deviceResults = append(deviceResults, deviceResult{
				Device:   device.Category,
				Viewport: device.Viewport,
				Status:   "failed",
				Error:    fmt.Sprintf("Failed to create temp dir: %v", err),
			})
			continue
		}

		// Build CLI args with device viewport
		args := []string{"scout", "--game", gameURL, "--json", "--save-flows", "--output", tmpDir, "--headless", "--timeout", "60"}
		if agentMode {
			args = append(args, "--agent")
			if req.AgentSteps > 0 {
				args = append(args, "--agent-steps", fmt.Sprintf("%d", req.AgentSteps))
			}
		}
		if req.Adaptive {
			args = append(args, "--adaptive")
			if req.MaxTotalSteps > 0 {
				args = append(args, "--max-total-steps", fmt.Sprintf("%d", req.MaxTotalSteps))
			}
		}
		if req.AdaptiveTimeout {
			args = append(args, "--adaptive-timeout")
			if req.MaxTotalTimeout > 0 {
				args = append(args, "--max-total-timeout", fmt.Sprintf("%d", req.MaxTotalTimeout))
			}
		}
		if req.Model != "" {
			args = append(args, "--model", req.Model)
		}
		if req.MaxTokens > 0 {
			args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxTokens))
		}
		if req.Temperature != nil {
			args = append(args, "--temperature", fmt.Sprintf("%g", *req.Temperature))
		}
		args = append(args, "--viewport", device.Viewport)
		if req.Modules.UIUX != nil && !*req.Modules.UIUX {
			args = append(args, "--no-uiux")
		}
		if req.Modules.Wording != nil && !*req.Modules.Wording {
			args = append(args, "--no-wording")
		}
		if req.Modules.GameDesign != nil && !*req.Modules.GameDesign {
			args = append(args, "--no-game-design")
		}
		if req.Modules.TestFlows != nil && !*req.Modules.TestFlows {
			args = append(args, "--no-test-flows")
		}

		log.Printf("Batch analysis %s [%s %d/%d]: executing %s %s", analysisID, device.Category, deviceNum, deviceTotal, cliPath, strings.Join(args, " "))
		cmd := exec.CommandContext(ctx, cliPath, args...)
		cmd.Env = append(os.Environ(), "NO_COLOR=1")

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			os.RemoveAll(tmpDir)
			deviceResults = append(deviceResults, deviceResult{
				Device:   device.Category,
				Viewport: device.Viewport,
				Status:   "failed",
				Error:    fmt.Sprintf("stdout pipe: %v", err),
			})
			continue
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			os.RemoveAll(tmpDir)
			deviceResults = append(deviceResults, deviceResult{
				Device:   device.Category,
				Viewport: device.Viewport,
				Status:   "failed",
				Error:    fmt.Sprintf("stderr pipe: %v", err),
			})
			continue
		}

		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			os.RemoveAll(tmpDir)
			deviceResults = append(deviceResults, deviceResult{
				Device:   device.Category,
				Viewport: device.Viewport,
				Status:   "failed",
				Error:    fmt.Sprintf("stdin pipe: %v", err),
			})
			continue
		}

		if err := cmd.Start(); err != nil {
			os.RemoveAll(tmpDir)
			deviceResults = append(deviceResults, deviceResult{
				Device:   device.Category,
				Viewport: device.Viewport,
				Status:   "failed",
				Error:    fmt.Sprintf("Failed to start CLI: %v", err),
			})
			continue
		}

		// Register active analysis for user→agent messaging
		s.activeAnalysesMu.Lock()
		s.activeAnalyses[analysisID] = &activeAnalysis{stdin: stdinPipe, tmpDir: tmpDir}
		s.activeAnalysesMu.Unlock()

		// Stream stderr for PROGRESS: lines, prefix messages with device label
		var stderrLines []string
		var lastKnownStep string
		stderrDone := make(chan struct{})
		go func() {
			defer close(stderrDone)
			stderrScanner := bufio.NewScanner(stderr)
			stderrScanner.Buffer(make([]byte, 256*1024), 256*1024)

			var latestReasoning string
			var lastStepDBID int64

			for stderrScanner.Scan() {
				line := stderrScanner.Text()
				if strings.HasPrefix(line, "PROGRESS:") {
					rest := line[len("PROGRESS:"):]
					parts := strings.SplitN(rest, ":", 2)
					step := strings.TrimSpace(parts[0])
					lastKnownStep = step
					message := ""
					if len(parts) > 1 {
						message = strings.TrimSpace(parts[1])
					}

					// Handle rich agent events
					switch step {
					case "agent_step_detail":
						var detailData map[string]interface{}
						if err := json.Unmarshal([]byte(message), &detailData); err == nil {
							detailData["analysisId"] = analysisID
							// Offset step numbers across devices
							if sn, ok := detailData["stepNumber"]; ok {
								if snFloat, ok := sn.(float64); ok {
									detailData["stepNumber"] = int(snFloat) + agentStepOffset
								}
							}
							s.wsHub.Broadcast(ws.Message{
								Type: "agent_step_detail",
								Data: detailData,
							})

							stepRecord := store.AgentStepRecord{
								AnalysisID: analysisID,
								StepNumber: intFromMap(detailData, "stepNumber"),
								ToolName:   strFromMap(detailData, "toolName"),
								Input:      strFromMap(detailData, "input"),
								Result:     strFromMap(detailData, "result"),
								DurationMs: intFromMap(detailData, "durationMs"),
								ThinkingMs: intFromMap(detailData, "thinkingMs"),
								Error:      strFromMap(detailData, "error"),
								Reasoning:  latestReasoning,
							}
							if dbID, saveErr := s.store.SaveAgentStep(stepRecord); saveErr != nil {
								log.Printf("Warning: failed to save agent step %d for %s: %v", stepRecord.StepNumber, analysisID, saveErr)
							} else {
								lastStepDBID = dbID
							}
							latestReasoning = ""
						}
					case "agent_reasoning":
						latestReasoning = message
						s.wsHub.Broadcast(ws.Message{
							Type: "agent_reasoning",
							Data: map[string]string{
								"analysisId": analysisID,
								"text":       message,
							},
						})
					case "agent_screenshot":
						filename := filepath.Base(message)
						if filename == "." || filename == "/" || strings.ContainsAny(filename, `/\`) {
							break
						}
						// Copy tmpDir under lock to avoid race with cleanup goroutine
						s.activeAnalysesMu.Lock()
						var aaTmpDir string
						if aa := s.activeAnalyses[analysisID]; aa != nil {
							aaTmpDir = aa.tmpDir
						}
						s.activeAnalysesMu.Unlock()
						if aaTmpDir != "" {
							screenshotPath := filepath.Join(aaTmpDir, "agent-screenshots", filename)
							if imgData, readErr := os.ReadFile(screenshotPath); readErr == nil {
								persisted := false
								dataDir := s.store.DataDir()
								if dataDir != "" {
									dstDir := filepath.Join(dataDir, "screenshots", analysisID)
									if mkErr := os.MkdirAll(dstDir, 0755); mkErr == nil {
										dstPath := filepath.Join(dstDir, filename)
										if cpErr := os.WriteFile(dstPath, imgData, 0644); cpErr == nil {
											persisted = true
											if lastStepDBID > 0 {
												s.store.UpdateAgentStepScreenshot(lastStepDBID, filename)
											}
										}
									}
								}
								if persisted {
									// Use direct filename-based URL to avoid DB lookup race
									s.wsHub.Broadcast(ws.Message{
										Type: "agent_screenshot",
										Data: map[string]string{
											"analysisId":    analysisID,
											"screenshotUrl": fmt.Sprintf("/api/analyses/%s/screenshots/%s", analysisID, filename),
											"filename":      filename,
										},
									})
								}
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

					// Prefix the message with device label for analysis_progress
					prefixedMessage := fmt.Sprintf("%s %s", deviceLabel, message)
					s.wsHub.Broadcast(ws.Message{
						Type: "analysis_progress",
						Data: AnalysisProgress{
							Step:    step,
							Message: prefixedMessage,
							Data: map[string]string{
								"analysisId":  analysisID,
								"device":      device.Category,
								"deviceIndex": fmt.Sprintf("%d", deviceNum),
								"deviceTotal": fmt.Sprintf("%d", deviceTotal),
							},
						},
					})
					go func(id, st string) {
						if err := s.store.UpdateAnalysisStatus(id, store.StatusRunning, st); err != nil {
							log.Printf("Warning: failed to update analysis %s step to %s: %v", id, st, err)
						}
					}(analysisID, step)
				} else {
					const maxStderrLines = 1000
					if len(stderrLines) >= maxStderrLines {
						stderrLines = stderrLines[1:]
					}
					stderrLines = append(stderrLines, line)
					log.Printf("Batch analysis %s [%s] stderr: %s", analysisID, device.Category, line)
				}
			}
		}()

		// Collect all stdout
		const maxOutputSize = 10 * 1024 * 1024
		var outputBuf bytes.Buffer
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			if outputBuf.Len() < maxOutputSize {
				outputBuf.WriteString(scanner.Text())
				outputBuf.WriteString("\n")
			}
		}

		<-stderrDone

		cmdErr := cmd.Wait()

		// Clean up active analysis registration.
		// Safe: cmd.Wait() above guarantees the subprocess has exited, so the stderr
		// goroutine (which also reads activeAnalyses) has completed via <-stderrDone.
		s.activeAnalysesMu.Lock()
		delete(s.activeAnalyses, analysisID)
		s.activeAnalysesMu.Unlock()
		stdinPipe.Close()

		if cmdErr != nil {
			var userMsg string
			if ctx.Err() != nil {
				userMsg = fmt.Sprintf("Timed out (last step: %s)", lastKnownStep)
			} else if exitErr, ok := cmdErr.(*exec.ExitError); ok {
				userMsg = fmt.Sprintf("CLI exited with code %d", exitErr.ExitCode())
				if lastKnownStep != "" {
					userMsg += fmt.Sprintf(" (failed during: %s)", lastKnownStep)
				}
			} else {
				userMsg = cmdErr.Error()
			}
			log.Printf("Batch analysis %s [%s]: device failed: %s", analysisID, device.Category, userMsg)

			os.RemoveAll(tmpDir)
			deviceResults = append(deviceResults, deviceResult{
				Device:   device.Category,
				Viewport: device.Viewport,
				Status:   "failed",
				Error:    userMsg,
			})
			continue
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
					os.RemoveAll(tmpDir)
					deviceResults = append(deviceResults, deviceResult{
						Device:   device.Category,
						Viewport: device.Viewport,
						Status:   "failed",
						Error:    fmt.Sprintf("Failed to parse CLI output: %v", err2),
					})
					continue
				}
			} else {
				os.RemoveAll(tmpDir)
				deviceResults = append(deviceResults, deviceResult{
					Device:   device.Category,
					Viewport: device.Viewport,
					Status:   "failed",
					Error:    fmt.Sprintf("Failed to parse CLI output: %v", err),
				})
				continue
			}
		}

		// Extract flows and prefix names with device category
		deviceFlowCount := 0
		if flows, ok := result["flows"]; ok {
			if flowSlice, ok := flows.([]interface{}); ok {
				for _, f := range flowSlice {
					if fMap, ok := f.(map[string]interface{}); ok {
						if name, ok := fMap["name"].(string); ok {
							fMap["name"] = device.Category + "_" + name
						}
						// Tag each flow with its device
						fMap["device"] = device.Category
						fMap["viewport"] = device.Viewport
					}
					allFlows = append(allFlows, f)
				}
				deviceFlowCount = len(flowSlice)
			}
		}

		// Collect agent steps with offset
		if steps, ok := result["agentSteps"]; ok {
			if stepSlice, ok := steps.([]interface{}); ok {
				for _, step := range stepSlice {
					if stepMap, ok := step.(map[string]interface{}); ok {
						if sn, ok := stepMap["stepNumber"].(float64); ok {
							stepMap["stepNumber"] = int(sn) + agentStepOffset
						}
						stepMap["device"] = device.Category
					}
					allAgentSteps = append(allAgentSteps, step)
				}
				agentStepOffset += len(stepSlice)
			}
		}

		// Use first device as primary for pageMeta and analysis
		if i == 0 {
			if pm, ok := result["pageMeta"].(map[string]interface{}); ok {
				primaryPageMeta = pm
			}
			if a, ok := result["analysis"].(map[string]interface{}); ok {
				primaryAnalysis = a
			}
			if m, ok := result["mode"].(string); ok {
				primaryMode = m
			}
		}

		// Extract game name and framework from first successful device
		if gameName == "" {
			if pm, ok := result["pageMeta"].(map[string]interface{}); ok {
				if t, ok := pm["title"].(string); ok {
					gameName = t
				}
				if f, ok := pm["framework"].(string); ok {
					framework = f
				}
			}
			if a, ok := result["analysis"].(map[string]interface{}); ok {
				if gi, ok := a["gameInfo"].(map[string]interface{}); ok {
					if n, ok := gi["name"].(string); ok && n != "" {
						gameName = n
					}
				}
			}
		}

		totalFlowCount += deviceFlowCount
		os.RemoveAll(tmpDir)

		deviceResults = append(deviceResults, deviceResult{
			Device:    device.Category,
			Viewport:  device.Viewport,
			FlowCount: deviceFlowCount,
			Status:    "completed",
		})

		log.Printf("Batch analysis %s [%s %d/%d]: completed with %d flows", analysisID, device.Category, deviceNum, deviceTotal, deviceFlowCount)
	}

	// Check if all devices failed
	allFailed := true
	for _, dr := range deviceResults {
		if dr.Status == "completed" {
			allFailed = false
			break
		}
	}

	if allFailed {
		// Collect all device errors for the message
		var errMsgs []string
		for _, dr := range deviceResults {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", dr.Device, dr.Error))
		}
		s.broadcastAnalysisError(analysisID, "All devices failed: "+strings.Join(errMsgs, "; "))
		return
	}

	// Build merged result
	mergedResult := map[string]interface{}{
		"flows":      allFlows,
		"agentSteps": allAgentSteps,
		"devices":    deviceResults,
	}
	if primaryPageMeta != nil {
		mergedResult["pageMeta"] = primaryPageMeta
	}
	if primaryAnalysis != nil {
		mergedResult["analysis"] = primaryAnalysis
	}
	if primaryMode != "" {
		mergedResult["mode"] = primaryMode
	}

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_progress",
		Data: AnalysisProgress{
			Step:    "saving",
			Message: "Saving generated flows...",
			Data:    map[string]string{"analysisId": analysisID},
		},
	})

	if err := s.store.UpdateAnalysisResult(analysisID, store.StatusCompleted, mergedResult, gameName, framework, totalFlowCount); err != nil {
		log.Printf("Warning: failed to update analysis record for %s: %v", analysisID, err)
	}

	// Record any partial failures
	for _, dr := range deviceResults {
		if dr.Status == "failed" {
			if saveErr := s.store.UpdateAnalysisError(analysisID, fmt.Sprintf("Device %s failed: %s", dr.Device, dr.Error)); saveErr != nil {
				log.Printf("Warning: failed to save device error for %s: %v", analysisID, saveErr)
			}
		}
	}

	// Auto-create one test plan with all flows
	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_progress",
		Data: AnalysisProgress{
			Step:    "test_plan",
			Message: fmt.Sprintf("Creating test plan from %d flows...", totalFlowCount),
			Data:    map[string]string{"analysisId": analysisID},
		},
	})

	testPlanID := s.autoCreateTestPlan(analysisID, gameURL, gameName, req.ProjectID, createdBy, totalFlowCount, req.AgentMode)

	if testPlanID != "" {
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "test_plan_done",
				Message: fmt.Sprintf("Test plan created: %s (%d flows)", gameName+" - Test Plan", totalFlowCount),
				Data:    map[string]string{"analysisId": analysisID, "testPlanId": testPlanID},
			},
		})
	} else if totalFlowCount > 0 {
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "test_plan_done",
				Message: "Test plan already exists for this analysis",
				Data:    map[string]string{"analysisId": analysisID},
			},
		})
	} else {
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "test_plan_done",
				Message: "No flows generated — test plan skipped",
				Data:    map[string]string{"analysisId": analysisID},
			},
		})
	}

	// TODO: Skip auto-run-tests for batch — viewport-per-flow routing is complex

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_completed",
		Data: map[string]interface{}{
			"analysisId": analysisID,
			"result":     mergedResult,
			"testPlanId": testPlanID,
		},
	})

	log.Printf("Batch analysis %s completed: %d devices, %d total flows", analysisID, len(req.Devices), totalFlowCount)
}

func (s *Server) executeAnalysis(analysisID, createdBy string, req AnalysisRequest) {
	gameURL := req.GameURL
	agentMode := req.AgentMode

	// Serialize modules to JSON for persistence
	modulesJSON := ""
	{
		m := map[string]bool{
			"uiux":      req.Modules.UIUX == nil || *req.Modules.UIUX,
			"wording":   req.Modules.Wording == nil || *req.Modules.Wording,
			"gameDesign": req.Modules.GameDesign == nil || *req.Modules.GameDesign,
			"testFlows": req.Modules.TestFlows == nil || *req.Modules.TestFlows,
			"runTests":  req.Modules.RunTests != nil && *req.Modules.RunTests,
		}
		if b, err := json.Marshal(m); err == nil {
			modulesJSON = string(b)
		}
	}

	// Serialize profile params for resume
	profileJSON := ""
	{
		p := map[string]interface{}{}
		if req.Model != "" {
			p["model"] = req.Model
		}
		if req.MaxTokens > 0 {
			p["maxTokens"] = req.MaxTokens
		}
		if req.Temperature != nil {
			p["temperature"] = *req.Temperature
		}
		if req.AgentSteps > 0 {
			p["agentSteps"] = req.AgentSteps
		}
		if req.Adaptive {
			p["adaptive"] = true
		}
		if req.MaxTotalSteps > 0 {
			p["maxTotalSteps"] = req.MaxTotalSteps
		}
		if req.AdaptiveTimeout {
			p["adaptiveTimeout"] = true
		}
		if req.MaxTotalTimeout > 0 {
			p["maxTotalTimeout"] = req.MaxTotalTimeout
		}
		if req.Viewport != "" {
			p["viewport"] = req.Viewport
		}
		if len(p) > 0 {
			if b, err := json.Marshal(p); err == nil {
				profileJSON = string(b)
			}
		}
	}

	// Save record BEFORE acquiring semaphore so status endpoint returns it while queued
	runningRecord := store.AnalysisRecord{
		ID:        analysisID,
		GameURL:   gameURL,
		Status:    store.StatusRunning,
		Step:      "queued",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
		CreatedBy: createdBy,
		ProjectID: req.ProjectID,
		Modules:   modulesJSON,
		AgentMode: agentMode,
		Profile:   profileJSON,
	}
	if err := s.store.SaveAnalysis(runningRecord); err != nil {
		log.Printf("Warning: failed to save running analysis record for %s: %v", analysisID, err)
	}

	// Acquire concurrency slot (released before inline test run to avoid two Chrome instances)
	// Try non-blocking first to avoid broadcasting "queued" when there's no contention
	select {
	case s.analysisSem <- struct{}{}:
		// Got it immediately
	default:
		// Semaphore is busy — notify user they're queued
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "queued",
				Message: "Another analysis is running. Waiting in queue...",
				Data:    map[string]string{"analysisId": analysisID},
			},
		})
		log.Printf("Analysis %s: queued (semaphore busy)", analysisID)
		queueTimeout := time.After(5 * time.Minute)
		select {
		case s.analysisSem <- struct{}{}:
			// Got it
		case <-queueTimeout:
			s.store.UpdateAnalysisStatus(analysisID, store.StatusFailed, "queue_timeout")
			s.broadcastAnalysisError(analysisID, "Timed out waiting in queue. Another analysis may be stuck — please try again.")
			return
		case <-s.serverCtx.Done():
			s.store.UpdateAnalysisStatus(analysisID, store.StatusFailed, "shutdown")
			s.broadcastAnalysisError(analysisID, "Server shutting down")
			return
		}
	}
	analysisSemHeld := true
	releaseAnalysisSem := func() {
		if analysisSemHeld {
			<-s.analysisSem
			analysisSemHeld = false
		}
	}
	defer releaseAnalysisSem()

	// Update step now that we have the semaphore
	if err := s.store.UpdateAnalysisStatus(analysisID, store.StatusRunning, "scouting"); err != nil {
		log.Printf("Warning: failed to update analysis %s step to scouting: %v", analysisID, err)
	}

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_progress",
		Data: AnalysisProgress{
			Step:    "scouting",
			Message: "Scouting page...",
			Data:    map[string]string{"analysisId": analysisID, "gameUrl": gameURL},
		},
	})

	cliPath := envOrDefault("WIZARDS_QA_CLI_PATH", "wizards-qa")

	tmpDir, err := os.MkdirTemp("", "wizards-qa-analysis-*")
	if err != nil {
		s.broadcastAnalysisError(analysisID, fmt.Sprintf("Failed to create temp dir: %v", err))
		return
	}
	defer os.RemoveAll(tmpDir)

	timeout := 5 * time.Minute
	if agentMode {
		steps := req.AgentSteps
		if steps <= 0 {
			steps = 20 // default
		}
		if req.Adaptive && req.MaxTotalSteps > steps {
			steps = req.MaxTotalSteps
		}
		// Base: exploration budget (steps × 75s avg) + 10min buffer for synthesis + flow gen (with retries)
		explorationBudget := time.Duration(steps) * 75 * time.Second
		timeout = explorationBudget + 10*time.Minute
		// When adaptive timeout is enabled, use maxTotalTimeout + buffer if it's larger
		if req.AdaptiveTimeout && req.MaxTotalTimeout > 0 {
			timeoutFromMinutes := time.Duration(req.MaxTotalTimeout)*time.Minute + 8*time.Minute
			if timeoutFromMinutes > timeout {
				timeout = timeoutFromMinutes
			}
		}
		// Clamp between 10min and 45min (60min for adaptive/adaptive-timeout)
		if timeout < 10*time.Minute {
			timeout = 10 * time.Minute
		}
		maxClamp := 45 * time.Minute
		if req.Adaptive || req.AdaptiveTimeout {
			maxClamp = 60 * time.Minute
		}
		if req.AdaptiveTimeout && req.MaxTotalTimeout > 0 {
			maxClamp = 60 * time.Minute
		}
		if timeout > maxClamp {
			timeout = maxClamp
		}
	}
	ctx, cancel := context.WithTimeout(s.serverCtx, timeout)
	defer cancel()

	args := []string{"scout", "--game", gameURL, "--json", "--save-flows", "--output", tmpDir, "--headless", "--timeout", "60"}
	if agentMode {
		args = append(args, "--agent")
		if req.AgentSteps > 0 {
			args = append(args, "--agent-steps", fmt.Sprintf("%d", req.AgentSteps))
		}
	}
	if req.Adaptive {
		args = append(args, "--adaptive")
		if req.MaxTotalSteps > 0 {
			args = append(args, "--max-total-steps", fmt.Sprintf("%d", req.MaxTotalSteps))
		}
	}
	if req.AdaptiveTimeout {
		args = append(args, "--adaptive-timeout")
		if req.MaxTotalTimeout > 0 {
			args = append(args, "--max-total-timeout", fmt.Sprintf("%d", req.MaxTotalTimeout))
		}
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxTokens))
	}
	if req.Temperature != nil {
		args = append(args, "--temperature", fmt.Sprintf("%g", *req.Temperature))
	}
	if req.Viewport != "" {
		args = append(args, "--viewport", req.Viewport)
	}
	if req.Modules.UIUX != nil && !*req.Modules.UIUX {
		args = append(args, "--no-uiux")
	}
	if req.Modules.Wording != nil && !*req.Modules.Wording {
		args = append(args, "--no-wording")
	}
	if req.Modules.GameDesign != nil && !*req.Modules.GameDesign {
		args = append(args, "--no-game-design")
	}
	if req.Modules.TestFlows != nil && !*req.Modules.TestFlows {
		args = append(args, "--no-test-flows")
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
	var lastKnownStep string
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		stderrScanner := bufio.NewScanner(stderr)
		stderrScanner.Buffer(make([]byte, 256*1024), 256*1024)

		// Track latest reasoning text and last inserted step ID for screenshot association
		var latestReasoning string
		var lastStepDBID int64

		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			if strings.HasPrefix(line, "PROGRESS:") {
				rest := line[len("PROGRESS:"):]
				parts := strings.SplitN(rest, ":", 2)
				step := strings.TrimSpace(parts[0])
				lastKnownStep = step
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

						// Persist step to database
						stepRecord := store.AgentStepRecord{
							AnalysisID: analysisID,
							StepNumber: intFromMap(detailData, "stepNumber"),
							ToolName:   strFromMap(detailData, "toolName"),
							Input:      strFromMap(detailData, "input"),
							Result:     strFromMap(detailData, "result"),
							DurationMs: intFromMap(detailData, "durationMs"),
							ThinkingMs: intFromMap(detailData, "thinkingMs"),
							Error:      strFromMap(detailData, "error"),
							Reasoning:  latestReasoning,
						}
						if dbID, saveErr := s.store.SaveAgentStep(stepRecord); saveErr != nil {
							log.Printf("Warning: failed to save agent step %d for %s: %v", stepRecord.StepNumber, analysisID, saveErr)
						} else {
							lastStepDBID = dbID
						}
						// Clear reasoning after attaching to a step
						latestReasoning = ""
					}
				case "agent_reasoning":
					latestReasoning = message
					s.wsHub.Broadcast(ws.Message{
						Type: "agent_reasoning",
						Data: map[string]string{
							"analysisId": analysisID,
							"text":       message,
						},
					})
				case "agent_screenshot":
					// Read screenshot file from tmpDir, persist to data dir, broadcast URL (not base64)
					filename := filepath.Base(message) // strip directory components
					if filename == "." || filename == "/" || strings.ContainsAny(filename, `/\`) {
						break
					}
					// Copy tmpDir under lock to avoid race with cleanup goroutine
					s.activeAnalysesMu.Lock()
					var aaTmpDir string
					if aa := s.activeAnalyses[analysisID]; aa != nil {
						aaTmpDir = aa.tmpDir
					}
					s.activeAnalysesMu.Unlock()
					if aaTmpDir != "" {
						screenshotPath := filepath.Join(aaTmpDir, "agent-screenshots", filename)
						if imgData, readErr := os.ReadFile(screenshotPath); readErr == nil {
							persisted := false
							dataDir := s.store.DataDir()
							if dataDir != "" {
								dstDir := filepath.Join(dataDir, "screenshots", analysisID)
								if mkErr := os.MkdirAll(dstDir, 0755); mkErr == nil {
									dstPath := filepath.Join(dstDir, filename)
									if cpErr := os.WriteFile(dstPath, imgData, 0644); cpErr == nil {
										persisted = true
										if lastStepDBID > 0 {
											s.store.UpdateAgentStepScreenshot(lastStepDBID, filename)
										}
									}
								}
							}
							if persisted {
								// Use direct filename-based URL to avoid DB lookup race
								s.wsHub.Broadcast(ws.Message{
									Type: "agent_screenshot",
									Data: map[string]string{
										"analysisId":    analysisID,
										"screenshotUrl": fmt.Sprintf("/api/analyses/%s/screenshots/%s", analysisID, filename),
										"filename":      filename,
									},
								})
							}
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
				// Cap at 1000 lines, rotating old lines out
				const maxStderrLines = 1000
				if len(stderrLines) >= maxStderrLines {
					stderrLines = stderrLines[1:]
				}
				stderrLines = append(stderrLines, line)
				log.Printf("Analysis %s stderr: %s", analysisID, line)
			}
		}
	}()

	// Collect all stdout (capped at 10MB to prevent OOM)
	const maxOutputSize = 10 * 1024 * 1024
	var outputBuf bytes.Buffer
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		if outputBuf.Len() < maxOutputSize {
			outputBuf.WriteString(scanner.Text())
			outputBuf.WriteString("\n")
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		log.Printf("Warning: scanner error reading analysis output for %s: %v", analysisID, scanErr)
	}

	<-stderrDone

	err = cmd.Wait()
	if err != nil {
		// Classify error concisely for the user
		var userMsg string
		if ctx.Err() != nil {
			if lastKnownStep != "" {
				userMsg = fmt.Sprintf("Analysis timed out after %d minutes (last step: %s)", int(timeout.Minutes()), lastKnownStep)
			} else {
				userMsg = fmt.Sprintf("Analysis timed out after %d minutes", int(timeout.Minutes()))
			}
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			if lastKnownStep != "" {
				userMsg = fmt.Sprintf("CLI exited with code %d (failed during: %s)", exitErr.ExitCode(), lastKnownStep)
			} else {
				userMsg = fmt.Sprintf("CLI exited with code %d", exitErr.ExitCode())
			}
			// Append last meaningful stderr line for context
			for i := len(stderrLines) - 1; i >= 0; i-- {
				if line := strings.TrimSpace(stderrLines[i]); line != "" {
					if len(line) > 200 {
						line = line[:200] + "..."
					}
					userMsg += "\n" + line
					break
				}
			}
		} else {
			userMsg = err.Error()
		}

		// Store full stderr in error_message column for debugging
		fullStderr := strings.Join(stderrLines, "\n")
		if saveErr := s.store.UpdateAnalysisError(analysisID, fullStderr); saveErr != nil {
			log.Printf("Warning: failed to save error_message for %s: %v", analysisID, saveErr)
		}

		// Read best checkpoint from tmpDir for resume capability
		hasCheckpoint := false
		if cpData := readBestCheckpoint(tmpDir); cpData != "" {
			hasCheckpoint = true
			if saveErr := s.store.UpdateAnalysisPartialResult(analysisID, cpData); saveErr != nil {
				log.Printf("Warning: failed to save partial_result for %s: %v", analysisID, saveErr)
			} else {
				log.Printf("Analysis %s: saved checkpoint for resume", analysisID)
			}
		}

		// Build stderrTail (last 10 lines)
		tailStart := len(stderrLines) - 10
		if tailStart < 0 {
			tailStart = 0
		}
		stderrTail := strings.Join(stderrLines[tailStart:], "\n")

		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}

		s.broadcastAnalysisError(analysisID, userMsg, map[string]interface{}{
			"exitCode":        exitCode,
			"lastStep":        lastKnownStep,
			"stderrLineCount": len(stderrLines),
			"hasCheckpoint":   hasCheckpoint,
			"stderrTail":      stderrTail,
		})
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

	// Save generated flows to disk for non-agent analyses (agent mode uses scenarios directly)
	if !agentMode {
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
	}

	// Count scenarios (agent mode) or flows (legacy mode) for the analysis record
	flowCount := countScenariosOrFlows(result, agentMode)

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

	// Auto-create a test plan
	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_progress",
		Data: AnalysisProgress{
			Step:    "test_plan",
			Message: fmt.Sprintf("Creating test plan from %d scenarios...", flowCount),
			Data:    map[string]string{"analysisId": analysisID},
		},
	})

	testPlanID := s.autoCreateTestPlan(analysisID, gameURL, gameName, req.ProjectID, createdBy, flowCount, agentMode)

	if testPlanID != "" {
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "test_plan_done",
				Message: fmt.Sprintf("Test plan created: %s (%d flows)", gameName+" - Test Plan", flowCount),
				Data:    map[string]string{"analysisId": analysisID, "testPlanId": testPlanID},
			},
		})
	} else if flowCount > 0 {
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "test_plan_done",
				Message: "Test plan already exists for this analysis",
				Data:    map[string]string{"analysisId": analysisID},
			},
		})
	} else {
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "test_plan_done",
				Message: "No flows generated — test plan skipped",
				Data:    map[string]string{"analysisId": analysisID},
			},
		})
	}

	// Release the analysis semaphore before running browser tests so that
	// the CLI Chrome process is fully dead before the test Chrome starts.
	releaseAnalysisSem()

	// Auto-run tests if enabled
	var testRunID string
	if req.Modules.RunTests != nil && *req.Modules.RunTests && testPlanID != "" {
		testMode := "agent"
		if !agentMode {
			testMode = "browser"
		}

		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "testing",
				Message: fmt.Sprintf("Running %s tests...", testMode),
				Data:    map[string]string{"analysisId": analysisID},
			},
		})

		plan, planErr := s.store.GetTestPlan(testPlanID)
		if planErr == nil {
			testRunID = newID("test")

			s.wsHub.Broadcast(ws.Message{
				Type: "analysis_progress",
				Data: AnalysisProgress{
					Step:    "testing_started",
					Message: fmt.Sprintf("Test run started (%s mode): %s", testMode, testRunID),
					Data: map[string]string{
						"analysisId": analysisID,
						"testId":     testRunID,
						"testPlanId": testPlanID,
					},
				},
			})

			viewport := req.Viewport
			if viewport == "" {
				viewport = "desktop-std"
			}

			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in auto-test for analysis %s: %v", analysisID, r)
				}
			}()

			if agentMode {
				s.executeAgentTestRun(testPlanID, testRunID, plan.AnalysisID, plan.Name, createdBy, viewport)
			} else {
				flowDir2, flowErr := s.prepareFlowDir(plan)
				if flowErr == nil {
					defer os.RemoveAll(flowDir2)
					s.executeBrowserTestRun(testPlanID, testRunID, flowDir2, plan.Name, createdBy, viewport)
				} else {
					log.Printf("Warning: failed to prepare flow dir for auto-test on %s: %v", analysisID, flowErr)
				}
			}

			s.wsHub.Broadcast(ws.Message{
				Type: "analysis_progress",
				Data: AnalysisProgress{
					Step:    "testing_done",
					Message: "Tests completed",
					Data:    map[string]string{"analysisId": analysisID, "testId": testRunID},
				},
			})

			if err := s.store.UpdateAnalysisTestRunID(analysisID, testRunID); err != nil {
				log.Printf("Warning: failed to update last_test_run_id for %s: %v", analysisID, err)
			}
		} else {
			log.Printf("Warning: failed to get test plan %s for auto-test: %v", testPlanID, planErr)
		}
	}

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_completed",
		Data: map[string]interface{}{
			"analysisId": analysisID,
			"result":     result,
			"testPlanId": testPlanID,
			"testRunId":  testRunID,
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

func intFromMap(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func strFromMap(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func (s *Server) broadcastAnalysisError(analysisID, errMsg string, extra ...map[string]interface{}) {
	log.Printf("Analysis %s failed: %s", analysisID, errMsg)

	if err := s.store.UpdateAnalysisStatus(analysisID, store.StatusFailed, ""); err != nil {
		log.Printf("Warning: failed to mark analysis %s as failed: %v", analysisID, err)
	}

	data := map[string]interface{}{
		"analysisId": analysisID,
		"error":      errMsg,
	}
	if len(extra) > 0 {
		for k, v := range extra[0] {
			data[k] = v
		}
	}
	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_failed",
		Data: data,
	})
}

// readBestCheckpoint reads the most advanced checkpoint file from a directory.
// Returns the JSON string or "" if no checkpoint found.
func readBestCheckpoint(dir string) string {
	for _, step := range []string{"synthesized", "analyzed", "scouted"} {
		path := filepath.Join(dir, fmt.Sprintf("checkpoint_%s.json", step))
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Validate it's parseable JSON
		var tmp map[string]interface{}
		if json.Unmarshal(data, &tmp) == nil {
			return string(data)
		}
	}
	return ""
}

func (s *Server) handleContinueAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	analysis, err := s.store.GetAnalysis(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Analysis not found")
		return
	}

	if analysis.Status != store.StatusFailed {
		respondError(w, http.StatusBadRequest, "Only failed analyses can be continued")
		return
	}

	if analysis.PartialResult == "" {
		respondError(w, http.StatusBadRequest, "No checkpoint data available — use retry instead")
		return
	}

	// Parse checkpoint to get step info
	var checkpoint map[string]interface{}
	if err := json.Unmarshal([]byte(analysis.PartialResult), &checkpoint); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid checkpoint data")
		return
	}
	step, _ := checkpoint["step"].(string)
	if step == "" {
		respondError(w, http.StatusBadRequest, "Checkpoint missing step info")
		return
	}

	// Get createdBy from auth context
	var createdBy string
	if claims := auth.UserFromContext(r.Context()); claims != nil {
		createdBy = claims.UserID
	}

	// Reset status to running
	if err := s.store.UpdateAnalysisStatus(id, store.StatusRunning, "resuming"); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update analysis status")
		return
	}

	// Clear error message
	if err := s.store.UpdateAnalysisError(id, ""); err != nil {
		log.Printf("Warning: failed to clear error_message for %s: %v", id, err)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in continued analysis %s: %v", id, r)
				s.broadcastAnalysisError(id, fmt.Sprintf("panic: %v", r))
			}
		}()
		s.executeContinuedAnalysis(id, createdBy, analysis)
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"analysisId": id,
		"status":     "continuing",
		"message":    fmt.Sprintf("Resuming from checkpoint (%s)", step),
	})
}

// autoCreateTestPlan creates a test plan linked to the analysis.
// For agent mode, it uses scenario names from the analysis result.
// For legacy mode, it uses flow filenames from the generated directory.
// Returns the plan ID or "" if skipped. Idempotent via GetTestPlanByAnalysis.
func (s *Server) autoCreateTestPlan(analysisID, gameURL, gameName, projectID, createdBy string, flowCount int, agentMode bool) string {
	if flowCount == 0 {
		return ""
	}

	// Idempotency: check if a plan already exists for this analysis
	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_progress",
		Data: AnalysisProgress{
			Step:    "test_plan_checking",
			Message: "Checking for existing test plan...",
			Data:    map[string]string{"analysisId": analysisID},
		},
	})
	existing, _ := s.store.GetTestPlanByAnalysis(analysisID)
	if existing != nil {
		return existing.ID
	}

	var flowNames []string
	planMode := ""

	if agentMode {
		// Extract scenario names from analysis result
		flowNames = s.extractScenarioNames(analysisID)
		planMode = "agent"
	} else {
		// Get flow filenames from the generated directory
		var err error
		flowNames, err = s.store.ListGeneratedFlowNames(analysisID)
		if err != nil || len(flowNames) == 0 {
			log.Printf("Warning: auto-create test plan skipped for %s: no generated flows found", analysisID)
			return ""
		}
	}

	if len(flowNames) == 0 {
		log.Printf("Warning: auto-create test plan skipped for %s: no scenarios/flows found", analysisID)
		return ""
	}

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_progress",
		Data: AnalysisProgress{
			Step:    "test_plan_flows",
			Message: fmt.Sprintf("Found %d scenarios: %s", len(flowNames), strings.Join(flowNames, ", ")),
			Data:    map[string]string{"analysisId": analysisID},
		},
	})

	planName := gameName
	if planName == "" {
		planName = "Analysis"
	}
	planName += " - Test Plan"

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_progress",
		Data: AnalysisProgress{
			Step:    "test_plan_saving",
			Message: fmt.Sprintf("Saving test plan: %s", planName),
			Data:    map[string]string{"analysisId": analysisID},
		},
	})

	plan := store.TestPlan{
		ID:         newID("plan"),
		Name:       planName,
		GameURL:    gameURL,
		FlowNames:  flowNames,
		Variables:  map[string]string{},
		Status:     store.StatusDraft,
		CreatedAt:  time.Now().Format(time.RFC3339),
		CreatedBy:  createdBy,
		ProjectID:  projectID,
		AnalysisID: analysisID,
		Mode:       planMode,
	}

	if err := s.store.SaveTestPlan(plan); err != nil {
		log.Printf("Warning: failed to auto-create test plan for analysis %s: %v", analysisID, err)
		return ""
	}

	log.Printf("Auto-created test plan %s for analysis %s (%d scenarios, mode=%s)", plan.ID, analysisID, len(flowNames), planMode)
	return plan.ID
}

// extractScenarioNames gets scenario names from the analysis result stored in DB.
func (s *Server) extractScenarioNames(analysisID string) []string {
	analysis, err := s.store.GetAnalysis(analysisID)
	if err != nil {
		return nil
	}
	resultMap, ok := analysis.Result.(map[string]interface{})
	if !ok {
		return nil
	}
	// Try analysis.scenarios first, then top-level scenarios
	analysisData, ok := resultMap["analysis"].(map[string]interface{})
	if !ok {
		analysisData = resultMap
	}
	scenariosRaw, ok := analysisData["scenarios"].([]interface{})
	if !ok {
		return nil
	}
	var names []string
	for _, s := range scenariosRaw {
		if m, ok := s.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok && name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

// countScenariosOrFlows counts scenarios (agent mode) or flows (legacy mode) from the CLI result.
func countScenariosOrFlows(result map[string]interface{}, agentMode bool) int {
	if agentMode {
		// Count scenarios from the analysis sub-object
		if analysisData, ok := result["analysis"].(map[string]interface{}); ok {
			if scenarios, ok := analysisData["scenarios"].([]interface{}); ok {
				return len(scenarios)
			}
		}
	}
	// Count flows (legacy)
	if flows, ok := result["flows"]; ok {
		if flowSlice, ok := flows.([]interface{}); ok {
			return len(flowSlice)
		}
	}
	return 0
}

func (s *Server) executeContinuedAnalysis(analysisID, createdBy string, analysis *store.AnalysisRecord) {
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
			Step:    "resuming",
			Message: "Resuming from checkpoint...",
			Data:    map[string]string{"analysisId": analysisID, "gameUrl": analysis.GameURL},
		},
	})

	cliPath := envOrDefault("WIZARDS_QA_CLI_PATH", "wizards-qa")

	tmpDir, err := os.MkdirTemp("", "wizards-qa-continue-*")
	if err != nil {
		s.broadcastAnalysisError(analysisID, fmt.Sprintf("Failed to create temp dir: %v", err))
		return
	}
	defer os.RemoveAll(tmpDir)

	// Write checkpoint data to tmpDir for CLI to read
	resumeDataPath := filepath.Join(tmpDir, "resume_data.json")
	if err := os.WriteFile(resumeDataPath, []byte(analysis.PartialResult), 0644); err != nil {
		s.broadcastAnalysisError(analysisID, fmt.Sprintf("Failed to write resume data: %v", err))
		return
	}

	// Parse checkpoint for step name
	var checkpoint map[string]interface{}
	json.Unmarshal([]byte(analysis.PartialResult), &checkpoint)
	step, _ := checkpoint["step"].(string)

	// Shorter timeout — only flow generation remaining
	timeout := 5 * time.Minute
	if analysis.AgentMode {
		timeout = 8 * time.Minute
		// Check if adaptive mode was enabled (profile may have longer timeout)
		if analysis.Profile != "" {
			var profile map[string]interface{}
			if json.Unmarshal([]byte(analysis.Profile), &profile) == nil {
				if adaptiveVal, ok := profile["adaptive"].(bool); ok && adaptiveVal {
					timeout = 10 * time.Minute
				}
				if atVal, ok := profile["adaptiveTimeout"].(bool); ok && atVal {
					timeout = 10 * time.Minute
				}
			}
		}
	}
	ctx, cancel := context.WithTimeout(s.serverCtx, timeout)
	defer cancel()

	args := []string{"scout", "--game", analysis.GameURL, "--json", "--save-flows", "--output", tmpDir, "--headless", "--timeout", "60"}
	args = append(args, "--resume-from", step, "--resume-data", resumeDataPath)

	if analysis.AgentMode {
		args = append(args, "--agent")
	}

	// Reconstruct profile params
	if analysis.Profile != "" {
		var profile map[string]interface{}
		if json.Unmarshal([]byte(analysis.Profile), &profile) == nil {
			if m, ok := profile["model"].(string); ok && m != "" {
				args = append(args, "--model", m)
			}
			if mt, ok := profile["maxTokens"].(float64); ok && mt > 0 {
				args = append(args, "--max-tokens", fmt.Sprintf("%d", int(mt)))
			}
			if t, ok := profile["temperature"].(float64); ok {
				args = append(args, "--temperature", fmt.Sprintf("%g", t))
			}
			if as, ok := profile["agentSteps"].(float64); ok && as > 0 {
				args = append(args, "--agent-steps", fmt.Sprintf("%d", int(as)))
			}
			if adaptiveVal, ok := profile["adaptive"].(bool); ok && adaptiveVal {
				args = append(args, "--adaptive")
			}
			if mts, ok := profile["maxTotalSteps"].(float64); ok && mts > 0 {
				args = append(args, "--max-total-steps", fmt.Sprintf("%d", int(mts)))
			}
			if atVal, ok := profile["adaptiveTimeout"].(bool); ok && atVal {
				args = append(args, "--adaptive-timeout")
			}
			if mtt, ok := profile["maxTotalTimeout"].(float64); ok && mtt > 0 {
				args = append(args, "--max-total-timeout", fmt.Sprintf("%d", int(mtt)))
			}
			if vp, ok := profile["viewport"].(string); ok && vp != "" {
				args = append(args, "--viewport", vp)
			}
		}
	}

	// Reconstruct module flags
	if analysis.Modules != "" {
		var mods map[string]bool
		if json.Unmarshal([]byte(analysis.Modules), &mods) == nil {
			if v, ok := mods["uiux"]; ok && !v {
				args = append(args, "--no-uiux")
			}
			if v, ok := mods["wording"]; ok && !v {
				args = append(args, "--no-wording")
			}
			if v, ok := mods["gameDesign"]; ok && !v {
				args = append(args, "--no-game-design")
			}
			if v, ok := mods["testFlows"]; ok && !v {
				args = append(args, "--no-test-flows")
			}
		}
	}

	log.Printf("Analysis %s: continuing with %s %s", analysisID, cliPath, strings.Join(args, " "))
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

	if err := cmd.Start(); err != nil {
		s.broadcastAnalysisError(analysisID, fmt.Sprintf("Failed to start CLI: %v", err))
		return
	}

	// Stream stderr for PROGRESS: lines
	var stderrLines []string
	var lastKnownStep string
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
				progressStep := strings.TrimSpace(parts[0])
				lastKnownStep = progressStep
				message := ""
				if len(parts) > 1 {
					message = strings.TrimSpace(parts[1])
				}
				s.wsHub.Broadcast(ws.Message{
					Type: "analysis_progress",
					Data: AnalysisProgress{
						Step:    progressStep,
						Message: message,
						Data:    map[string]string{"analysisId": analysisID},
					},
				})
				go func(id, st string) {
					if err := s.store.UpdateAnalysisStatus(id, store.StatusRunning, st); err != nil {
						log.Printf("Warning: failed to update analysis %s step to %s: %v", id, st, err)
					}
				}(analysisID, progressStep)
			} else {
				// Cap at 1000 lines, rotating old lines out
				const maxStderrLines = 1000
				if len(stderrLines) >= maxStderrLines {
					stderrLines = stderrLines[1:]
				}
				stderrLines = append(stderrLines, line)
				log.Printf("Analysis %s stderr: %s", analysisID, line)
			}
		}
	}()

	// Collect all stdout (capped at 10MB to prevent OOM)
	const maxOutputSize = 10 * 1024 * 1024
	var outputBuf bytes.Buffer
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		if outputBuf.Len() < maxOutputSize {
			outputBuf.WriteString(scanner.Text())
			outputBuf.WriteString("\n")
		}
	}

	<-stderrDone

	err = cmd.Wait()
	if err != nil {
		var userMsg string
		if ctx.Err() != nil {
			if lastKnownStep != "" {
				userMsg = fmt.Sprintf("Continued analysis timed out after %d minutes (last step: %s)", int(timeout.Minutes()), lastKnownStep)
			} else {
				userMsg = fmt.Sprintf("Continued analysis timed out after %d minutes", int(timeout.Minutes()))
			}
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			userMsg = fmt.Sprintf("CLI exited with code %d", exitErr.ExitCode())
			for i := len(stderrLines) - 1; i >= 0; i-- {
				if line := strings.TrimSpace(stderrLines[i]); line != "" {
					if len(line) > 200 {
						line = line[:200] + "..."
					}
					userMsg += "\n" + line
					break
				}
			}
		} else {
			userMsg = err.Error()
		}

		fullStderr := strings.Join(stderrLines, "\n")
		if saveErr := s.store.UpdateAnalysisError(analysisID, fullStderr); saveErr != nil {
			log.Printf("Warning: failed to save error_message for %s: %v", analysisID, saveErr)
		}

		// Update checkpoint in case a later one was written
		hasCheckpoint := false
		if cpData := readBestCheckpoint(tmpDir); cpData != "" {
			hasCheckpoint = true
			if saveErr := s.store.UpdateAnalysisPartialResult(analysisID, cpData); saveErr != nil {
				log.Printf("Warning: failed to update partial_result for %s: %v", analysisID, saveErr)
			}
		}

		// Build stderrTail (last 10 lines)
		tailStart := len(stderrLines) - 10
		if tailStart < 0 {
			tailStart = 0
		}
		stderrTail := strings.Join(stderrLines[tailStart:], "\n")

		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}

		s.broadcastAnalysisError(analysisID, userMsg, map[string]interface{}{
			"exitCode":        exitCode,
			"lastStep":        lastKnownStep,
			"stderrLineCount": len(stderrLines),
			"hasCheckpoint":   hasCheckpoint,
			"stderrTail":      stderrTail,
		})
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

	// Save generated flows to disk for non-agent analyses (agent mode uses scenarios directly)
	if !analysis.AgentMode {
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
	}

	// Count scenarios (agent mode) or flows (legacy mode) for the analysis record
	flowCount := countScenariosOrFlows(result, analysis.AgentMode)

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
	if a, ok := result["analysis"].(map[string]interface{}); ok {
		if gi, ok := a["gameInfo"].(map[string]interface{}); ok {
			if n, ok := gi["name"].(string); ok && n != "" {
				gameName = n
			}
		}
	}

	// Clear partial_result on success
	if err := s.store.UpdateAnalysisPartialResult(analysisID, ""); err != nil {
		log.Printf("Warning: failed to clear partial_result for %s: %v", analysisID, err)
	}

	if err := s.store.UpdateAnalysisResult(analysisID, store.StatusCompleted, result, gameName, framework, flowCount); err != nil {
		log.Printf("Warning: failed to update analysis record for %s: %v", analysisID, err)
	}

	// Auto-create a test plan
	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_progress",
		Data: AnalysisProgress{
			Step:    "test_plan",
			Message: fmt.Sprintf("Creating test plan from %d scenarios...", flowCount),
			Data:    map[string]string{"analysisId": analysisID},
		},
	})

	testPlanID := s.autoCreateTestPlan(analysisID, analysis.GameURL, gameName, analysis.ProjectID, createdBy, flowCount, analysis.AgentMode)

	if testPlanID != "" {
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "test_plan_done",
				Message: fmt.Sprintf("Test plan created: %s (%d flows)", gameName+" - Test Plan", flowCount),
				Data:    map[string]string{"analysisId": analysisID, "testPlanId": testPlanID},
			},
		})
	} else if flowCount > 0 {
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "test_plan_done",
				Message: "Test plan already exists for this analysis",
				Data:    map[string]string{"analysisId": analysisID},
			},
		})
	} else {
		s.wsHub.Broadcast(ws.Message{
			Type: "analysis_progress",
			Data: AnalysisProgress{
				Step:    "test_plan_done",
				Message: "No flows generated — test plan skipped",
				Data:    map[string]string{"analysisId": analysisID},
			},
		})
	}

	s.wsHub.Broadcast(ws.Message{
		Type: "analysis_completed",
		Data: map[string]interface{}{
			"analysisId": analysisID,
			"result":     result,
			"testPlanId": testPlanID,
		},
	})
}
