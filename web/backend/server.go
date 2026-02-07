package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type Server struct {
	router *chi.Mux
	port   string
}

func NewServer(port string) *Server {
	s := &Server{
		router: chi.NewRouter(),
		port:   port,
	}
	s.setupMiddleware()
	s.setupRoutes()
	return s
}

func (s *Server) setupMiddleware() {
	// CORS
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Logging
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
		})
	})
}

func (s *Server) setupRoutes() {
	s.router.Get("/api/health", s.handleHealth)
	s.router.Get("/api/tests", s.handleListTests)
	s.router.Get("/api/tests/{id}", s.handleGetTest)
	s.router.Post("/api/tests/run", s.handleRunTest)
	s.router.Get("/api/reports", s.handleListReports)
	s.router.Get("/api/reports/{id}", s.handleGetReport)
	s.router.Get("/api/flows", s.handleListFlows)
	s.router.Get("/api/flows/{name}", s.handleGetFlow)
	s.router.Get("/api/stats", s.handleGetStats)
	
	// Serve frontend static files
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "web/frontend/dist"))
	FileServer(s.router, "/", filesDir)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleListTests(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement - query test database/files
	tests := []map[string]interface{}{
		{
			"id":         "test-001",
			"name":       "Simple Platformer",
			"status":     "passed",
			"timestamp":  time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			"duration":   "45s",
			"successRate": 100.0,
		},
		{
			"id":         "test-002",
			"name":       "Puzzle Game",
			"status":     "failed",
			"timestamp":  time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			"duration":   "32s",
			"successRate": 75.0,
		},
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tests": tests,
		"total": len(tests),
	})
}

func (s *Server) handleGetTest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	// TODO: Implement - query specific test
	test := map[string]interface{}{
		"id":          id,
		"name":        "Simple Platformer",
		"status":      "passed",
		"timestamp":   time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		"duration":    "45s",
		"successRate": 100.0,
		"flows": []map[string]interface{}{
			{
				"name":     "launch.yaml",
				"status":   "passed",
				"duration": "5.2s",
			},
			{
				"name":     "gameplay.yaml",
				"status":   "passed",
				"duration": "12.3s",
			},
		},
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
	
	// TODO: Implement - execute wizards-qa test command
	testID := fmt.Sprintf("test-%d", time.Now().Unix())
	
	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"testId":  testID,
		"status":  "running",
		"message": "Test execution started",
	})
}

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement - list report files
	reports := []map[string]interface{}{
		{
			"id":        "report-001",
			"name":      "Simple Platformer Test",
			"format":    "markdown",
			"timestamp": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			"size":      "12.5 KB",
		},
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"reports": reports,
		"total":   len(reports),
	})
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	// TODO: Implement - read report file
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"content": "# Test Report\n\n...",
	})
}

func (s *Server) handleListFlows(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement - scan flows directory
	flows := []map[string]interface{}{
		{
			"name":     "click-object",
			"category": "game-mechanics",
			"path":     "flows/templates/game-mechanics/click-object.yaml",
		},
		{
			"name":     "collect-items",
			"category": "game-mechanics",
			"path":     "flows/templates/game-mechanics/collect-items.yaml",
		},
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"flows": flows,
		"total": len(flows),
	})
}

func (s *Server) handleGetFlow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	
	// TODO: Implement - read flow file
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"name":    name,
		"content": "url: https://example.com\n---\n...",
	})
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement - aggregate statistics
	stats := map[string]interface{}{
		"totalTests":    25,
		"passedTests":   20,
		"failedTests":   5,
		"avgDuration":   "42s",
		"avgSuccessRate": 85.3,
		"recentTests": []map[string]interface{}{
			{
				"name":       "Simple Platformer",
				"status":     "passed",
				"timestamp":  time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			},
		},
	}
	
	respondJSON(w, http.StatusOK, stats)
}

func (s *Server) Start() error {
	log.Printf("🧙‍♂️ Wizards QA Dashboard starting on http://localhost:%s\n", s.port)
	return http.ListenAndServe(":"+s.port, s.router)
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := NewServer(port)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
