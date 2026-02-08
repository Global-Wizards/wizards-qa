package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/Global-Wizards/wizards-qa/web/backend/auth"
	"github.com/Global-Wizards/wizards-qa/web/backend/store"
	"github.com/Global-Wizards/wizards-qa/web/backend/ws"
)

// Version is set at build time via -ldflags "-X main.Version=...".
// Defaults to "dev" for local development; see VERSION file for the release version.
var Version = "dev"

type Server struct {
	router       *chi.Mux
	port         string
	store        *store.Store
	wsHub        *ws.Hub
	jwtSecret    string
	serverCtx    context.Context
	cancelCtx    context.CancelFunc
	analysisSem  chan struct{} // limits concurrent analyses
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
		router:      chi.NewRouter(),
		port:        port,
		store:       st,
		wsHub:       hub,
		jwtSecret:   jwtSecret,
		serverCtx:   ctx,
		cancelCtx:   cancel,
		analysisSem: make(chan struct{}, 3), // max 3 concurrent analyses
	}
	s.setupMiddleware()
	s.setupRoutes()
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
		r.Post("/api/tests/run", s.handleRunTest)
		r.Get("/api/reports", s.handleListReports)
		r.Get("/api/reports/{id}", s.handleGetReport)
		r.Get("/api/flows", s.handleListFlows)
		r.Get("/api/flows/{name}", s.handleGetFlow)
		r.Get("/api/stats", s.handleGetStats)
		r.Get("/api/config", s.handleGetConfig)
		r.Get("/api/performance", s.handleGetPerformance)
		r.Get("/api/templates", s.handleListTemplates)
		r.Get("/api/test-plans", s.handleListTestPlans)
		r.Post("/api/test-plans", s.handleCreateTestPlan)
		r.Get("/api/test-plans/{id}", s.handleGetTestPlan)
		r.Post("/api/test-plans/{id}/run", s.handleRunTestPlan)
		r.Delete("/api/test-plans/{id}", s.handleDeleteTestPlan)
		r.Post("/api/analyze", s.handleAnalyzeGame)
		r.Get("/api/analyses", s.handleListAnalyses)
		r.Get("/api/analyses/{id}", s.handleGetAnalysis)
		r.Get("/api/analyses/{id}/status", s.handleGetAnalysisStatus)
		r.Delete("/api/analyses/{id}", s.handleDeleteAnalysis)
		r.Get("/api/analyses/{id}/export", s.handleExportAnalysis)

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

func (s *Server) handleRunTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameURL  string `json:"gameUrl"`
		SpecPath string `json:"specPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
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

	plan.ID = fmt.Sprintf("plan-%d", time.Now().UnixNano())
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
	respondJSON(w, http.StatusOK, plan)
}

func (s *Server) handleRunTestPlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	plan, err := s.store.GetTestPlan(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Test plan not found")
		return
	}

	flowDir, err := s.prepareFlowDir(plan)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to prepare flows: "+err.Error())
		return
	}

	testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	var createdByPlan string
	if claims := auth.UserFromContext(r.Context()); claims != nil {
		createdByPlan = claims.UserID
	}
	s.launchTestRun(plan.ID, testID, flowDir, plan.Name, true, createdByPlan)

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"testId":  testID,
		"planId":  plan.ID,
		"status":  store.StatusRunning,
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
	respondJSON(w, http.StatusOK, analysis)
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
		}
	}

	sb.WriteString("\n---\n*Generated by Wizards QA*\n")
	return sb.String()
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

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
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
			indexFile, indexErr := root.Open("/index.html")
			if indexErr != nil {
				http.NotFound(w, r)
				return
			}
			defer indexFile.Close()
			stat, _ := indexFile.Stat()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile)
			return
		}
		f.Close()

		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
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
}
