package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// InitDB opens (or creates) the SQLite database and runs migrations.
func InitDB(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Set pragmas for performance and correctness
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma %q: %w", p, err)
		}
	}

	if err := createTables(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating tables: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			game_url TEXT NOT NULL DEFAULT '',
			description TEXT DEFAULT '',
			color TEXT DEFAULT '#6366f1',
			icon TEXT DEFAULT 'gamepad-2',
			tags TEXT DEFAULT '[]',
			settings TEXT DEFAULT '{}',
			created_by TEXT REFERENCES users(id),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS project_members (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role TEXT NOT NULL DEFAULT 'member',
			created_at TEXT NOT NULL,
			UNIQUE(project_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			display_name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS analyses (
			id TEXT PRIMARY KEY,
			game_url TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'running',
			step TEXT DEFAULT '',
			framework TEXT DEFAULT '',
			game_name TEXT DEFAULT '',
			flow_count INTEGER DEFAULT 0,
			result TEXT,
			created_by TEXT REFERENCES users(id),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS test_results (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			duration TEXT DEFAULT '',
			success_rate REAL DEFAULT 0,
			flows TEXT,
			error_output TEXT DEFAULT '',
			created_by TEXT REFERENCES users(id),
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS test_plans (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			game_url TEXT DEFAULT '',
			flow_names TEXT,
			variables TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			last_run_id TEXT DEFAULT '',
			created_by TEXT REFERENCES users(id),
			created_at TEXT NOT NULL
		)`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("executing %q: %w", s[:40], err)
		}
	}
	return nil
}

// runMigrations adds project_id columns to existing tables idempotently.
func runMigrations(db *sql.DB) error {
	// Add project_id columns — ignore errors for duplicate columns
	alters := []string{
		`ALTER TABLE analyses ADD COLUMN project_id TEXT DEFAULT ''`,
		`ALTER TABLE test_plans ADD COLUMN project_id TEXT DEFAULT ''`,
		`ALTER TABLE test_results ADD COLUMN project_id TEXT DEFAULT ''`,
	}
	for _, stmt := range alters {
		if _, err := db.Exec(stmt); err != nil {
			// Ignore "duplicate column" errors
			if !strings.Contains(err.Error(), "duplicate column") {
				log.Printf("Migration warning (non-fatal): %v", err)
			}
		}
	}

	// Create indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_analyses_project ON analyses(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_test_plans_project ON test_plans(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_test_results_project ON test_results(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_project_members_project ON project_members(project_id)`,
	}
	for _, stmt := range indexes {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("creating index: %w", err)
		}
	}

	return nil
}

// MigrateToProjects auto-creates projects from existing game_url data.
// Idempotent: only runs when unassigned records exist.
func (s *Store) MigrateToProjects() {
	var unassigned int
	s.db.QueryRow(`SELECT COUNT(*) FROM analyses WHERE project_id = '' AND game_url != ''`).Scan(&unassigned)
	var unassignedPlans int
	s.db.QueryRow(`SELECT COUNT(*) FROM test_plans WHERE project_id = '' AND game_url != ''`).Scan(&unassignedPlans)

	if unassigned == 0 && unassignedPlans == 0 {
		return
	}

	// Collect distinct game_urls
	urls := make(map[string]bool)
	rows, err := s.db.Query(`SELECT DISTINCT game_url FROM analyses WHERE project_id = '' AND game_url != ''`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var u string
			if rows.Scan(&u) == nil {
				urls[u] = true
			}
		}
	}
	rows2, err := s.db.Query(`SELECT DISTINCT game_url FROM test_plans WHERE project_id = '' AND game_url != ''`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var u string
			if rows2.Scan(&u) == nil {
				urls[u] = true
			}
		}
	}

	created := 0
	now := time.Now().Format(time.RFC3339)

	for gameURL := range urls {
		// Derive project name from hostname
		name := gameURL
		if parsed, err := neturl.Parse(gameURL); err == nil && parsed.Host != "" {
			name = parsed.Host
		}

		projectID := fmt.Sprintf("proj-%d-%d", time.Now().UnixNano(), created)

		_, err := s.db.Exec(
			`INSERT INTO projects (id, name, game_url, description, color, icon, tags, settings, created_at, updated_at)
			 VALUES (?, ?, ?, '', '#6366f1', 'gamepad-2', '[]', '{}', ?, ?)`,
			projectID, name, gameURL, now, now,
		)
		if err != nil {
			log.Printf("MigrateToProjects: failed to create project for %s: %v", gameURL, err)
			continue
		}

		// Assign analyses
		if _, err := s.db.Exec(`UPDATE analyses SET project_id = ? WHERE game_url = ? AND project_id = ''`, projectID, gameURL); err != nil {
			log.Printf("MigrateToProjects: failed to assign analyses for %s: %v", gameURL, err)
		}
		// Assign test plans
		if _, err := s.db.Exec(`UPDATE test_plans SET project_id = ? WHERE game_url = ? AND project_id = ''`, projectID, gameURL); err != nil {
			log.Printf("MigrateToProjects: failed to assign test plans for %s: %v", gameURL, err)
		}
		// Assign test results via test plan's last_run_id
		if _, err := s.db.Exec(`UPDATE test_results SET project_id = ? WHERE id IN (
			SELECT last_run_id FROM test_plans WHERE project_id = ? AND last_run_id != ''
		)`, projectID, projectID); err != nil {
			log.Printf("MigrateToProjects: failed to assign test results for project %s: %v", projectID, err)
		}

		created++
	}

	if created > 0 {
		log.Printf("MigrateToProjects: auto-created %d project(s) from existing game_url data", created)
	}
}

// MigrateFromJSON performs a one-time migration of existing JSON data files into SQLite.
func (s *Store) MigrateFromJSON(dataDir string) {
	// Only migrate if DB tables are empty
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM analyses").Scan(&count)
	if count > 0 {
		return // DB already has data
	}

	s.migrateAnalyses(dataDir)
	s.migrateTestResults(dataDir)
	s.migrateTestPlans(dataDir)
}

func (s *Store) migrateAnalyses(dataDir string) {
	path := filepath.Join(dataDir, "analyses.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var file AnalysesFile
	if err := json.Unmarshal(data, &file); err != nil {
		log.Printf("Migration: failed to parse analyses.json: %v", err)
		return
	}

	migrated := 0
	for _, a := range file.Analyses {
		var resultJSON *string
		if a.Result != nil {
			b, err := json.Marshal(a.Result)
			if err == nil {
				s := string(b)
				resultJSON = &s
			}
		}
		now := time.Now().Format(time.RFC3339)
		createdAt := a.CreatedAt
		if createdAt == "" {
			createdAt = now
		}
		updatedAt := a.UpdatedAt
		if updatedAt == "" {
			updatedAt = now
		}

		_, err := s.db.Exec(
			`INSERT OR IGNORE INTO analyses (id, game_url, status, step, framework, game_name, flow_count, result, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			a.ID, a.GameURL, a.Status, a.Step, a.Framework, a.GameName, a.FlowCount, resultJSON, createdAt, updatedAt,
		)
		if err != nil {
			log.Printf("Migration: failed to insert analysis %s: %v", a.ID, err)
			continue
		}
		migrated++
	}
	if migrated > 0 {
		log.Printf("Migration: migrated %d analyses from JSON", migrated)
	}
}

func (s *Store) migrateTestResults(dataDir string) {
	path := filepath.Join(dataDir, "test-results.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var file TestResultsFile
	if err := json.Unmarshal(data, &file); err != nil {
		// Try plain array
		var results []TestResultDetail
		if err2 := json.Unmarshal(data, &results); err2 != nil {
			log.Printf("Migration: failed to parse test-results.json: %v", err)
			return
		}
		file.Results = results
	}

	migrated := 0
	for _, r := range file.Results {
		var flowsJSON *string
		if r.Flows != nil {
			b, err := json.Marshal(r.Flows)
			if err == nil {
				s := string(b)
				flowsJSON = &s
			}
		}
		ts := r.Timestamp
		if ts == "" {
			ts = time.Now().Format(time.RFC3339)
		}

		_, err := s.db.Exec(
			`INSERT OR IGNORE INTO test_results (id, name, status, timestamp, duration, success_rate, flows, error_output, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.ID, r.Name, r.Status, ts, r.Duration, r.SuccessRate, flowsJSON, r.ErrorOutput, ts,
		)
		if err != nil {
			log.Printf("Migration: failed to insert test result %s: %v", r.ID, err)
			continue
		}
		migrated++
	}
	if migrated > 0 {
		log.Printf("Migration: migrated %d test results from JSON", migrated)
	}
}

func (s *Store) migrateTestPlans(dataDir string) {
	path := filepath.Join(dataDir, "test-plans.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var file TestPlansFile
	if err := json.Unmarshal(data, &file); err != nil {
		log.Printf("Migration: failed to parse test-plans.json: %v", err)
		return
	}

	migrated := 0
	for _, p := range file.Plans {
		var flowNamesJSON *string
		if p.FlowNames != nil {
			b, _ := json.Marshal(p.FlowNames)
			s := string(b)
			flowNamesJSON = &s
		}
		var variablesJSON *string
		if p.Variables != nil {
			b, _ := json.Marshal(p.Variables)
			s := string(b)
			variablesJSON = &s
		}
		createdAt := p.CreatedAt
		if createdAt == "" {
			createdAt = time.Now().Format(time.RFC3339)
		}

		_, err := s.db.Exec(
			`INSERT OR IGNORE INTO test_plans (id, name, description, game_url, flow_names, variables, status, last_run_id, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.ID, p.Name, p.Description, p.GameURL, flowNamesJSON, variablesJSON, p.Status, p.LastRunID, createdAt,
		)
		if err != nil {
			log.Printf("Migration: failed to insert test plan %s: %v", p.ID, err)
			continue
		}
		migrated++
	}
	if migrated > 0 {
		log.Printf("Migration: migrated %d test plans from JSON", migrated)
	}
}
