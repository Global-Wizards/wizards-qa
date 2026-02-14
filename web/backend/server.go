package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/Global-Wizards/wizards-qa/pkg/flows"
	"github.com/Global-Wizards/wizards-qa/web/backend/auth"
	"github.com/Global-Wizards/wizards-qa/web/backend/store"
	"github.com/Global-Wizards/wizards-qa/web/backend/ws"
)

// Version is set at build time via -ldflags "-X main.Version=...".
// Defaults to "dev" for local development; see VERSION file for the release version.
var Version = "dev"

// activeAnalysis tracks a running analysis subprocess for user→agent messaging.
type activeAnalysis struct {
	stdin       io.WriteCloser
	tmpDir      string
	lastHintAt  time.Time
}

type Server struct {
	router           *chi.Mux
	port             string
	store            *store.Store
	wsHub            *ws.Hub
	jwtSecret        string
	serverCtx        context.Context
	cancelCtx        context.CancelFunc
	analysisSem      chan struct{} // limits concurrent analyses
	browserTestSem   chan struct{} // limits concurrent browser test runs
	activeAnalyses   map[string]*activeAnalysis
	activeAnalysesMu sync.Mutex
	runningTests *RunningTestTracker
}

func NewServer(port string) *Server {
	flowsDir := envOrDefault("WIZARDS_QA_FLOWS_DIR", "flows/templates")
	reportsDir := envOrDefault("WIZARDS_QA_REPORTS_DIR", "reports")
	dataDir := envOrDefault("WIZARDS_QA_DATA_DIR", "data")
	configPath := envOrDefault("WIZARDS_QA_CONFIG", "wizards-qa.yaml")

	// Initialize SQLite database
	dbPath := filepath.Join(dataDir, "wizards.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	st := store.New(db, flowsDir, reportsDir, configPath)
	st.SetDataDir(dataDir)

	// Validate directories on startup
	if err := st.ValidateDirectories(); err != nil {
		log.Printf("Warning: directory validation failed: %v", err)
	}

	// One-time migration from JSON files to SQLite
	st.MigrateFromJSON(dataDir)

	// Recover orphaned running test plans from previous crash
	st.RecoverOrphanedRuns()

	// Recover orphaned running analyses from previous crash
	st.RecoverOrphanedAnalyses()

	// Auto-migrate existing data to projects
	st.MigrateToProjects()

	// JWT secret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Generate a random secret and log a warning
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			log.Fatal("Failed to generate JWT secret")
		}
		jwtSecret = hex.EncodeToString(b)
		log.Printf("Warning: JWT_SECRET not set, using random secret (tokens will not survive restarts)")
	}

	hub := ws.NewHub()
	go hub.Run()

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		router:         chi.NewRouter(),
		port:           port,
		store:          st,
		wsHub:          hub,
		jwtSecret:      jwtSecret,
		serverCtx:      ctx,
		cancelCtx:      cancel,
		analysisSem:    make(chan struct{}, 1), // max 1 concurrent analysis (Chrome uses 200-400MB)
		browserTestSem: make(chan struct{}, 1), // max 1 concurrent browser test run
		activeAnalyses: make(map[string]*activeAnalysis),
		runningTests:   NewRunningTestTracker(),
	}
	s.setupMiddleware()
	s.setupRoutes()

	// Periodic cleanup of stale runningTests entries (e.g. from crashes/timeouts)
	go s.cleanupStaleRunningTests()

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*", "https://*.fly.dev"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(sw, r)
			userID := "-"
			if claims := auth.UserFromContext(r.Context()); claims != nil {
				userID = claims.UserID
			}
			log.Printf("%s %s %d %s user=%s", r.Method, r.URL.Path, sw.status, time.Since(start), userID)
		})
	})
}

func (s *Server) setupRoutes() {
	// Rate limiter for auth endpoints (10 requests per minute per IP)
	authLimiter := newRateLimiter(10, time.Minute)

	// Public routes (no auth)
	s.router.Group(func(r chi.Router) {
		r.Use(authLimiter.Middleware)
		r.Post("/api/auth/register", s.handleRegister)
		r.Post("/api/auth/login", s.handleLogin)
		r.Post("/api/auth/refresh", s.handleRefresh)
	})
	s.router.Get("/api/health", s.handleHealth)
	s.router.Get("/api/version", s.handleVersion)
	s.router.Get("/api/changelog", s.handleChangelog)

	// Protected routes (require auth)
	s.router.Group(func(r chi.Router) {
		r.Use(auth.Middleware(s.jwtSecret))

		r.Get("/api/auth/me", s.handleMe)
		r.Get("/api/tests", s.handleListTests)
		r.Get("/api/tests/{id}", s.handleGetTest)
		r.Get("/api/tests/{id}/live", s.handleGetLiveTest)
		r.Post("/api/tests/run", s.handleRunTest)
		r.Delete("/api/tests/{id}", s.handleDeleteTestResult)
		r.Post("/api/tests/delete-batch", s.handleDeleteTestResultsBatch)
		r.Get("/api/reports", s.handleListReports)
		r.Get("/api/reports/{id}", s.handleGetReport)
		r.Get("/api/flows", s.handleListFlows)
		r.Get("/api/flows/{name}", s.handleGetFlow)
		r.Post("/api/flows/validate", s.handleValidateFlow)
		r.Get("/api/stats", s.handleGetStats)
		r.Get("/api/config", s.handleGetConfig)
		r.Get("/api/performance", s.handleGetPerformance)
		r.Get("/api/templates", s.handleListTemplates)
		r.Get("/api/test-plans", s.handleListTestPlans)
		r.Post("/api/test-plans", s.handleCreateTestPlan)
		r.Get("/api/test-plans/{id}", s.handleGetTestPlan)
		r.Put("/api/test-plans/{id}", s.handleUpdateTestPlan)
		r.Post("/api/test-plans/{id}/run", s.handleRunTestPlan)
		r.Delete("/api/test-plans/{id}", s.handleDeleteTestPlan)
		r.Post("/api/analyze", s.handleAnalyzeGame)
		r.Post("/api/analyze/batch", s.handleBatchAnalyze)
		r.Get("/api/analyses", s.handleListAnalyses)
		r.Get("/api/analyses/{id}", s.handleGetAnalysis)
		r.Get("/api/analyses/{id}/status", s.handleGetAnalysisStatus)
		r.Delete("/api/analyses/{id}", s.handleDeleteAnalysis)
		r.Get("/api/analyses/{id}/export", s.handleExportAnalysis)
		r.Get("/api/analyses/{id}/flows", s.handleListAnalysisFlows)
		r.Get("/api/analyses/{id}/steps", s.handleListAgentSteps)
		r.Get("/api/analyses/{id}/steps/{stepNumber}/screenshot", s.handleAgentStepScreenshot)
		r.Get("/api/analyses/{id}/screenshots/{filename}", s.handleAnalysisScreenshot)
		r.Post("/api/analyses/{id}/message", s.handleSendAgentMessage)
		r.Post("/api/analyses/{id}/continue", s.handleContinueAnalysis)
		r.Get("/api/tests/{testId}/steps/{flowName}/{stepIndex}/screenshot", s.handleTestStepScreenshot)

		// Project routes
		r.Get("/api/projects", s.handleListProjects)
		r.Post("/api/projects", s.handleCreateProject)
		r.Route("/api/projects/{projectId}", func(pr chi.Router) {
			pr.Get("/", s.handleGetProject)
			pr.Put("/", s.handleUpdateProject)
			pr.Delete("/", s.handleDeleteProject)
			pr.Get("/stats", s.handleGetProjectStats)
			pr.Get("/analyses", s.handleListProjectAnalyses)
			pr.Get("/test-plans", s.handleListProjectTestPlans)
			pr.Get("/tests", s.handleListProjectTests)
			pr.Get("/members", s.handleListProjectMembers)
			pr.Post("/members", s.handleAddProjectMember)
			pr.Put("/members/{userId}", s.handleUpdateMemberRole)
			pr.Delete("/members/{userId}", s.handleRemoveProjectMember)
		})
	})

	// WebSocket — validate token from query param ?token=...
	s.router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(s.wsHub, w, r, s.jwtSecret)
	})

	// Serve frontend static files
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "web/frontend/dist"))
	FileServer(s.router, "/", filesDir)
}

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Deep health check if ?deep=true
	if r.URL.Query().Get("deep") == "true" {
		checks := map[string]string{}
		overall := "ok"

		// DB check
		if err := s.store.Ping(); err != nil {
			checks["database"] = "error: " + err.Error()
			overall = "degraded"
		} else {
			checks["database"] = "ok"
		}

		// CLI check
		cliPath := envOrDefault("WIZARDS_QA_CLI_PATH", "wizards-qa")
		if _, err := os.Stat(cliPath); err != nil {
			checks["cli"] = "missing"
			overall = "degraded"
		} else {
			checks["cli"] = "ok"
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":  overall,
			"version": Version,
			"time":    time.Now().Format(time.RFC3339),
			"checks":  checks,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"version": Version,
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"version": Version,
		"name":    "Wizards QA",
	})
}

func (s *Server) handleChangelog(w http.ResponseWriter, r *http.Request) {
	// Try common locations for the changelog file
	candidates := []string{"CHANGELOG.md", "../CHANGELOG.md", "../../CHANGELOG.md"}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"content": string(data),
				"version": Version,
			})
			return
		}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"content": "# Changelog\n\nNo changelog available.",
		"version": Version,
	})
}

func (s *Server) handleListTests(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r, 50)
	tests, err := s.store.ListTestResults(limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list tests")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tests":  nonNil(tests),
		"total":  len(tests),
		"limit":  limit,
		"offset": offset,
	})
}

func (s *Server) handleGetTest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	test, err := s.store.GetTestResult(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Test not found")
		return
	}
	respondJSON(w, http.StatusOK, test)
}

func (s *Server) handleGetLiveTest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check running tests first
	if rt := s.runningTests.Get(id); rt != nil {
		respondJSON(w, http.StatusOK, rt)
		return
	}

	// Fall back to completed result
	test, err := s.store.GetTestResult(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Test not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"testId":      test.ID,
		"planId":      test.PlanID,
		"planName":    test.Name,
		"status":      test.Status,
		"flows":       test.Flows,
		"totalFlows":  len(test.Flows),
		"duration":    test.Duration,
		"successRate": test.SuccessRate,
		"errorOutput": test.ErrorOutput,
	})
}

func (s *Server) handleRunTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameURL  string `json:"gameUrl"`
		SpecPath string `json:"specPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	testID := newID("test")
	flowDir := s.store.FlowsDir()
	if req.SpecPath != "" {
		flowDir = filepath.Dir(req.SpecPath)
	}

	var createdBy string
	if claims := auth.UserFromContext(r.Context()); claims != nil {
		createdBy = claims.UserID
	}
	s.launchTestRun("", testID, flowDir, filepath.Base(flowDir), false, createdBy)

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"testId":  testID,
		"status":  store.StatusRunning,
		"message": "Test execution started",
	})
}

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := s.store.ListReports()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list reports")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"reports": nonNil(reports),
		"total":   len(reports),
	})
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	report, err := s.store.GetReport(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Report not found")
		return
	}
	respondJSON(w, http.StatusOK, report)
}

func (s *Server) handleListFlows(w http.ResponseWriter, r *http.Request) {
	flows, err := s.store.ListFlows()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list flows")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"flows": nonNil(flows),
		"total": len(flows),
	})
}

func (s *Server) handleGetFlow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	flow, err := s.store.GetFlow(name)
	if err != nil {
		respondError(w, http.StatusNotFound, "Flow not found")
		return
	}
	respondJSON(w, http.StatusOK, flow)
}

func (s *Server) handleValidateFlow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	result := validateMaestroYAML(req.Content)

	// If validation failed, try normalizing and re-validating.
	// If normalized version is better, offer it as a suggested fix.
	if !result.Valid {
		normalized := flows.NormalizeFlowYAML(req.Content)
		if normalized != req.Content {
			fixResult := validateMaestroYAML(normalized)
			if fixResult.Valid || len(fixResult.Errors) < len(result.Errors) {
				result.NormalizedContent = normalized
			}
		}
	}

	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.GetStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := s.store.GetConfig()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to read config")
		return
	}
	respondJSON(w, http.StatusOK, config)
}

func (s *Server) handleGetPerformance(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"uptime":          time.Since(startTime).String(),
		"activeWsClients": s.wsHub.ClientCount(),
		"version":         Version,
	})
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.store.ListTemplates()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list templates")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"templates": nonNil(templates),
		"total":     len(templates),
	})
}

func (s *Server) handleListTestPlans(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r, 50)
	plans, err := s.store.ListTestPlans(limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list test plans")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"plans":  nonNil(plans),
		"total":  len(plans),
		"limit":  limit,
		"offset": offset,
	})
}

func (s *Server) handleCreateTestPlan(w http.ResponseWriter, r *http.Request) {
	var plan store.TestPlan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if plan.Name == "" {
		respondError(w, http.StatusBadRequest, "Plan name is required")
		return
	}
	if len(plan.Name) > 200 {
		respondError(w, http.StatusBadRequest, "Plan name must be 200 characters or less")
		return
	}
	if plan.GameURL != "" {
		if _, err := url.ParseRequestURI(plan.GameURL); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid game URL")
			return
		}
	}

	plan.ID = newID("plan")
	plan.Status = store.StatusDraft
	plan.CreatedAt = time.Now().Format(time.RFC3339)
	if plan.Variables == nil {
		plan.Variables = make(map[string]string)
	}
	// ProjectID is accepted from the request body (already decoded above)

	// Set created_by from auth context
	if claims := auth.UserFromContext(r.Context()); claims != nil {
		plan.CreatedBy = claims.UserID
	}

	if err := s.store.SaveTestPlan(plan); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to save test plan")
		return
	}

	respondJSON(w, http.StatusCreated, plan)
}

func (s *Server) handleGetTestPlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	plan, err := s.store.GetTestPlan(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Test plan not found")
		return
	}

	if r.URL.Query().Get("include") == "flows" {
		type flowEntry struct {
			Name    string `json:"name"`
			Content string `json:"content"`
			Error   string `json:"error,omitempty"`
		}
		var flows []flowEntry
		needsRegen := false
		for _, name := range plan.FlowNames {
			fd, err := s.store.GetFlow(name)
			if err != nil {
				needsRegen = true
				flows = append(flows, flowEntry{Name: name, Error: err.Error()})
			} else {
				flows = append(flows, flowEntry{Name: fd.Name, Content: fd.Content})
			}
		}
		// If any flows were missing and the plan is linked to an analysis,
		// try regenerating from the analysis result (handles ephemeral storage loss).
		if needsRegen && plan.AnalysisID != "" {
			if rErr := s.regenerateFlowsFromAnalysis(plan.AnalysisID); rErr != nil {
				log.Printf("Warning: could not regenerate flows for analysis %s: %v", plan.AnalysisID, rErr)
			} else {
				// Retry loading all flows after regeneration
				flows = flows[:0]
				for _, name := range plan.FlowNames {
					fd, err := s.store.GetFlow(name)
					if err != nil {
						flows = append(flows, flowEntry{Name: name, Error: err.Error()})
					} else {
						flows = append(flows, flowEntry{Name: fd.Name, Content: fd.Content})
					}
				}
			}
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"plan":  plan,
			"flows": nonNil(flows),
		})
		return
	}

	respondJSON(w, http.StatusOK, plan)
}

func (s *Server) handleUpdateTestPlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := s.store.GetTestPlan(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Test plan not found")
		return
	}

	// Admin or owner check
	claims := auth.UserFromContext(r.Context())
	if claims != nil && claims.Role != "admin" {
		if existing.CreatedBy != "" && existing.CreatedBy != claims.UserID {
			respondError(w, http.StatusForbidden, "Only the owner or an admin can update this test plan")
			return
		}
	}

	var req struct {
		Name         string            `json:"name"`
		Description  string            `json:"description"`
		GameURL      string            `json:"gameUrl"`
		FlowNames    []string          `json:"flowNames"`
		Variables    map[string]string `json:"variables"`
		FlowContents map[string]string `json:"flowContents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Plan name is required")
		return
	}
	if len(req.Name) > 200 {
		respondError(w, http.StatusBadRequest, "Plan name must be 200 characters or less")
		return
	}
	if req.GameURL != "" {
		if _, err := url.ParseRequestURI(req.GameURL); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid game URL")
			return
		}
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.GameURL = req.GameURL
	existing.FlowNames = req.FlowNames
	if req.Variables != nil {
		existing.Variables = req.Variables
	}

	if err := s.store.UpdateTestPlan(*existing); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update test plan")
		return
	}

	var flowWarnings []string
	for name, content := range req.FlowContents {
		if err := s.store.SaveFlowContent(name, content); err != nil {
			flowWarnings = append(flowWarnings, fmt.Sprintf("%s: %s", name, err.Error()))
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"plan":         existing,
		"flowWarnings": nonNil(flowWarnings),
	})
}

func (s *Server) handleRunTestPlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	plan, err := s.store.GetTestPlan(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Test plan not found")
		return
	}

	// Parse optional run mode and viewport from request body
	var req struct {
		Mode     string `json:"mode"`     // "maestro" (default), "browser", or "agent"
		Viewport string `json:"viewport"` // viewport preset name for browser/agent mode
	}
	// Body is optional — ignore decode errors for backward compat (e.g. empty body)
	json.NewDecoder(r.Body).Decode(&req)

	testID := newID("test")
	var createdByPlan string
	if claims := auth.UserFromContext(r.Context()); claims != nil {
		createdByPlan = claims.UserID
	}

	mode := req.Mode
	if mode == "" {
		mode = plan.Mode
	}
	switch mode {
	case "agent":
		if plan.AnalysisID == "" {
			respondError(w, http.StatusBadRequest, "Agent mode requires an analysis-linked plan")
			return
		}
		viewport := req.Viewport
		if viewport == "" {
			viewport = "desktop-std"
		}
		s.launchAgentTestRun(plan.ID, testID, plan.AnalysisID, plan.Name, viewport, createdByPlan)

	case "browser":
		flowDir, err := s.prepareFlowDir(plan)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to prepare flows: "+err.Error())
			return
		}
		viewport := req.Viewport
		if viewport == "" {
			viewport = "desktop-std"
		}
		s.launchBrowserTestRun(plan.ID, testID, flowDir, plan.Name, viewport, true, createdByPlan)

	default:
		mode = "maestro"
		flowDir, err := s.prepareFlowDir(plan)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to prepare flows: "+err.Error())
			return
		}
		s.launchTestRun(plan.ID, testID, flowDir, plan.Name, true, createdByPlan)
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"testId":  testID,
		"planId":  plan.ID,
		"status":  store.StatusRunning,
		"mode":    mode,
		"message": "Test execution started",
	})
}

func (s *Server) handleListAnalyses(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r, 50)
	analyses, err := s.store.ListAnalyses(limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list analyses")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"analyses": nonNil(analyses),
		"total":    len(analyses),
		"limit":    limit,
		"offset":   offset,
	})
}

func (s *Server) handleGetAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	analysis, err := s.store.GetAnalysis(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Analysis not found")
		return
	}

	// Enrich with linked test plan ID if one exists
	resp := map[string]interface{}{
		"id":            analysis.ID,
		"gameUrl":       analysis.GameURL,
		"status":        analysis.Status,
		"step":          analysis.Step,
		"framework":     analysis.Framework,
		"gameName":      analysis.GameName,
		"flowCount":     analysis.FlowCount,
		"result":        analysis.Result,
		"createdBy":     analysis.CreatedBy,
		"projectId":     analysis.ProjectID,
		"createdAt":     analysis.CreatedAt,
		"updatedAt":     analysis.UpdatedAt,
		"errorMessage":  analysis.ErrorMessage,
		"modules":       analysis.Modules,
		"partialResult": analysis.PartialResult,
		"agentMode":     analysis.AgentMode,
		"profile":       analysis.Profile,
		"lastTestRunId": analysis.LastTestRunID,
	}

	if plan, _ := s.store.GetTestPlanByAnalysis(id); plan != nil {
		resp["testPlanId"] = plan.ID
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetAnalysisStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	analysis, err := s.store.GetAnalysis(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Analysis not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"id":     analysis.ID,
		"status": analysis.Status,
		"step":   analysis.Step,
	})
}

func (s *Server) handleDeleteAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Admin or owner check
	claims := auth.UserFromContext(r.Context())
	if claims != nil && claims.Role != "admin" {
		analysis, err := s.store.GetAnalysis(id)
		if err != nil {
			respondError(w, http.StatusNotFound, "Analysis not found")
			return
		}
		if analysis.CreatedBy != "" && analysis.CreatedBy != claims.UserID {
			respondError(w, http.StatusForbidden, "Only the owner or an admin can delete this analysis")
			return
		}
	}

	if err := s.store.DeleteAnalysis(id); err != nil {
		respondError(w, http.StatusNotFound, "Analysis not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleExportAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	analysis, err := s.store.GetAnalysis(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Analysis not found")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.json"`, id))
		json.NewEncoder(w).Encode(analysis)

	case "markdown":
		w.Header().Set("Content-Type", "text/markdown")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.md"`, id))
		md := formatAnalysisMarkdown(analysis)
		w.Write([]byte(md))

	default:
		respondError(w, http.StatusBadRequest, "Unsupported format: use json or markdown")
	}
}

func formatAnalysisMarkdown(a *store.AnalysisRecord) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Analysis: %s\n\n", a.GameName))
	sb.WriteString(fmt.Sprintf("- **URL:** %s\n", a.GameURL))
	sb.WriteString(fmt.Sprintf("- **Framework:** %s\n", a.Framework))
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", a.Status))
	sb.WriteString(fmt.Sprintf("- **Flows Generated:** %d\n", a.FlowCount))
	sb.WriteString(fmt.Sprintf("- **Date:** %s\n", a.CreatedAt))

	if result, ok := a.Result.(map[string]interface{}); ok {
		if analysis, ok := result["analysis"].(map[string]interface{}); ok {
			if mechanics, ok := analysis["mechanics"].([]interface{}); ok && len(mechanics) > 0 {
				sb.WriteString("\n## Mechanics\n\n")
				for _, m := range mechanics {
					if mMap, ok := m.(map[string]interface{}); ok {
						sb.WriteString(fmt.Sprintf("- **%v**: %v\n", mMap["name"], mMap["description"]))
					}
				}
			}
			if uiElements, ok := analysis["uiElements"].([]interface{}); ok && len(uiElements) > 0 {
				sb.WriteString("\n## UI Elements\n\n")
				for _, el := range uiElements {
					if elMap, ok := el.(map[string]interface{}); ok {
						sb.WriteString(fmt.Sprintf("- **%v** (%v): %v\n", elMap["name"], elMap["type"], elMap["selector"]))
					}
				}
			}
			if userFlows, ok := analysis["userFlows"].([]interface{}); ok && len(userFlows) > 0 {
				sb.WriteString("\n## User Flows\n\n")
				for _, f := range userFlows {
					if fMap, ok := f.(map[string]interface{}); ok {
						sb.WriteString(fmt.Sprintf("- **%v**: %v\n", fMap["name"], fMap["description"]))
					}
				}
			}
			if edgeCases, ok := analysis["edgeCases"].([]interface{}); ok && len(edgeCases) > 0 {
				sb.WriteString("\n## Edge Cases\n\n")
				for _, ec := range edgeCases {
					if ecMap, ok := ec.(map[string]interface{}); ok {
						sb.WriteString(fmt.Sprintf("- **%v**: %v\n", ecMap["name"], ecMap["description"]))
					}
				}
			}
			if uiux, ok := analysis["uiuxAnalysis"].([]interface{}); ok && len(uiux) > 0 {
				sb.WriteString("\n## UI/UX Analysis\n\n")
				for _, item := range uiux {
					if m, ok := item.(map[string]interface{}); ok {
						sb.WriteString(fmt.Sprintf("- [%v / %v] %v", m["severity"], m["category"], m["description"]))
						if sug, ok := m["suggestion"].(string); ok && sug != "" {
							sb.WriteString(fmt.Sprintf(" — *%v*", sug))
						}
						sb.WriteString("\n")
					}
				}
			}
			if wording, ok := analysis["wordingCheck"].([]interface{}); ok && len(wording) > 0 {
				sb.WriteString("\n## Wording/Translation Check\n\n")
				for _, item := range wording {
					if m, ok := item.(map[string]interface{}); ok {
						sb.WriteString(fmt.Sprintf("- [%v / %v]", m["severity"], m["category"]))
						if text, ok := m["text"].(string); ok && text != "" {
							sb.WriteString(fmt.Sprintf(" \"%v\"", text))
						}
						sb.WriteString(fmt.Sprintf(" — %v", m["description"]))
						if sug, ok := m["suggestion"].(string); ok && sug != "" {
							sb.WriteString(fmt.Sprintf(" → *%v*", sug))
						}
						sb.WriteString("\n")
					}
				}
			}
			if gd, ok := analysis["gameDesign"].([]interface{}); ok && len(gd) > 0 {
				sb.WriteString("\n## Game Design Analysis\n\n")
				for _, item := range gd {
					if m, ok := item.(map[string]interface{}); ok {
						sb.WriteString(fmt.Sprintf("- [%v / %v] %v", m["severity"], m["category"], m["description"]))
						if impact, ok := m["impact"].(string); ok && impact != "" {
							sb.WriteString(fmt.Sprintf(" (Impact: %v)", impact))
						}
						if sug, ok := m["suggestion"].(string); ok && sug != "" {
							sb.WriteString(fmt.Sprintf(" — *%v*", sug))
						}
						sb.WriteString("\n")
					}
				}
			}
		}
	}

	sb.WriteString("\n---\n*Generated by Wizards QA*\n")
	return sb.String()
}

func (s *Server) handleListAnalysisFlows(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	flowNames, err := s.store.ListGeneratedFlowNames(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list flows")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"flowNames": nonNil(flowNames),
	})
}

func (s *Server) handleListAgentSteps(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	steps, err := s.store.ListAgentSteps(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list agent steps")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"steps": nonNil(steps),
		"total": len(steps),
	})
}

func (s *Server) handleAgentStepScreenshot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	stepNumberStr := chi.URLParam(r, "stepNumber")

	stepNumber, err := strconv.Atoi(stepNumberStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid step number")
		return
	}

	screenshotPath, err := s.store.GetAgentStepScreenshot(id, stepNumber)
	if err != nil || screenshotPath == "" {
		respondError(w, http.StatusNotFound, "Screenshot not found")
		return
	}

	dataDir := s.store.DataDir()
	if dataDir == "" {
		respondError(w, http.StatusNotFound, "Screenshot storage not configured")
		return
	}

	// Sanitize path to prevent directory traversal
	screenshotPath = filepath.Base(screenshotPath)
	fullPath := filepath.Join(dataDir, "screenshots", id, screenshotPath)

	imgData, err := os.ReadFile(fullPath)
	if err != nil {
		respondError(w, http.StatusNotFound, "Screenshot file not found")
		return
	}

	// Detect content type from file extension (backward compat with existing .jpg screenshots)
	if strings.HasSuffix(screenshotPath, ".webp") {
		w.Header().Set("Content-Type", "image/webp")
	} else {
		w.Header().Set("Content-Type", "image/jpeg")
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(imgData)
}

// handleAnalysisScreenshot serves a screenshot by filename directly from disk,
// without requiring a DB lookup. Used for live screenshots during analysis.
func (s *Server) handleAnalysisScreenshot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	filename := chi.URLParam(r, "filename")

	// Sanitize to prevent directory traversal
	filename = filepath.Base(filename)
	if filename == "." || filename == "/" || filename == "" {
		respondError(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	dataDir := s.store.DataDir()
	if dataDir == "" {
		respondError(w, http.StatusNotFound, "Screenshot storage not configured")
		return
	}

	fullPath := filepath.Join(dataDir, "screenshots", id, filename)
	imgData, err := os.ReadFile(fullPath)
	if err != nil {
		respondError(w, http.StatusNotFound, "Screenshot file not found")
		return
	}

	if strings.HasSuffix(filename, ".webp") {
		w.Header().Set("Content-Type", "image/webp")
	} else {
		w.Header().Set("Content-Type", "image/jpeg")
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(imgData)
}

func (s *Server) handleDeleteTestPlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Admin or owner check
	claims := auth.UserFromContext(r.Context())
	if claims != nil && claims.Role != "admin" {
		plan, err := s.store.GetTestPlan(id)
		if err != nil {
			respondError(w, http.StatusNotFound, "Test plan not found")
			return
		}
		if plan.CreatedBy != "" && plan.CreatedBy != claims.UserID {
			respondError(w, http.StatusForbidden, "Only the owner or an admin can delete this test plan")
			return
		}
	}

	if err := s.store.DeleteTestPlan(id); err != nil {
		respondError(w, http.StatusNotFound, "Test plan not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleDeleteTestResult(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteTestResult(id); err != nil {
		respondError(w, http.StatusNotFound, "Test result not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleDeleteTestResultsBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		respondError(w, http.StatusBadRequest, "Missing or empty ids array")
		return
	}
	deleted, err := s.store.DeleteTestResultsBatch(req.IDs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete test results")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"deleted": deleted})
}

// --- Server lifecycle ---

// NewHTTPServer creates a configured *http.Server from the Server's router.
func (s *Server) NewHTTPServer() *http.Server {
	return &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// --- Helpers ---

// newID generates a unique ID with the given prefix and a random suffix for collision resistance.
func newID(prefix string) string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UnixNano(), hex.EncodeToString(b))
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// Hijack implements http.Hijacker so WebSocket upgrades work through the
// logging middleware. Without this, gorilla/websocket fails with
// "response does not implement http.Hijacker".
func (sw *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := sw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// parsePagination extracts limit and offset from query params with defaults.
func parsePagination(r *http.Request, defaultLimit int) (limit, offset int) {
	limit = defaultLimit
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// FileServer serves static files from a http.FileSystem with SPA fallback.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		requestPath := strings.TrimPrefix(r.URL.Path, pathPrefix)
		if requestPath == "" {
			requestPath = "/"
		}

		f, err := root.Open(requestPath)
		if err != nil {
			// SPA fallback: serve index.html with no-cache so deployments take effect immediately
			indexFile, indexErr := root.Open("/index.html")
			if indexErr != nil {
				http.NotFound(w, r)
				return
			}
			defer indexFile.Close()
			stat, _ := indexFile.Stat()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile)
			return
		}
		f.Close()

		// Hashed assets (JS/CSS with content hash) can be cached indefinitely
		if strings.Contains(requestPath, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

// cleanupStaleRunningTests periodically removes runningTests entries older than 30 minutes.
// This prevents memory leaks from test runs that crash or timeout without calling finishTestRun.
func (s *Server) cleanupStaleRunningTests() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for id, rt := range s.runningTests.GetAll() {
				if time.Since(rt.StartedAt) > 30*time.Minute {
					log.Printf("Cleaning up stale running test %s (started %s ago)", id, time.Since(rt.StartedAt).Round(time.Second))
					s.runningTests.Remove(id)
				}
			}
		case <-s.serverCtx.Done():
			return
		}
	}
}

var startTime = time.Now()

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := NewServer(port)
	srv := server.NewHTTPServer()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down gracefully...", sig)
		server.cancelCtx()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Printf("Wizards QA Dashboard v%s starting on http://localhost:%s", Version, port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	// Clean up database connection
	if err := server.store.Close(); err != nil {
		log.Printf("Warning: failed to close database: %v", err)
	}
}
