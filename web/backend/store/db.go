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

	// SQLite only supports one writer at a time. Limiting the pool to a single
	// connection ensures PRAGMAs (which are per-connection) persist across all
	// operations and eliminates SQLITE_BUSY errors from concurrent goroutines
	// getting separate pool connections without busy_timeout set.
	db.SetMaxOpenConns(1)

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
		`CREATE TABLE IF NOT EXISTS agent_steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			analysis_id TEXT NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
			step_number INTEGER NOT NULL,
			tool_name TEXT NOT NULL,
			input TEXT DEFAULT '',
			result TEXT DEFAULT '',
			screenshot_path TEXT DEFAULT '',
			duration_ms INTEGER DEFAULT 0,
			error TEXT DEFAULT '',
			reasoning TEXT DEFAULT '',
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
		`ALTER TABLE analyses ADD COLUMN error_message TEXT DEFAULT ''`,
		`ALTER TABLE analyses ADD COLUMN modules TEXT DEFAULT ''`,
		`ALTER TABLE analyses ADD COLUMN partial_result TEXT DEFAULT ''`,
		`ALTER TABLE analyses ADD COLUMN agent_mode INTEGER DEFAULT 0`,
		`ALTER TABLE analyses ADD COLUMN profile TEXT DEFAULT ''`,
		`ALTER TABLE test_plans ADD COLUMN analysis_id TEXT DEFAULT ''`,
		`ALTER TABLE test_results ADD COLUMN plan_id TEXT DEFAULT ''`,
		`ALTER TABLE agent_steps ADD COLUMN thinking_ms INTEGER DEFAULT 0`,
		`ALTER TABLE analyses ADD COLUMN last_test_run_id TEXT DEFAULT ''`,
		`ALTER TABLE test_plans ADD COLUMN mode TEXT DEFAULT ''`,
		`ALTER TABLE analyses ADD COLUMN total_credits INTEGER DEFAULT 0`,
		`ALTER TABLE analyses ADD COLUMN input_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE analyses ADD COLUMN output_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE analyses ADD COLUMN api_call_count INTEGER DEFAULT 0`,
		`ALTER TABLE analyses ADD COLUMN ai_model TEXT DEFAULT ''`,
		`ALTER TABLE agent_steps ADD COLUMN input_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE agent_steps ADD COLUMN output_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE agent_steps ADD COLUMN credits INTEGER DEFAULT 0`,
		`ALTER TABLE test_results ADD COLUMN total_credits INTEGER DEFAULT 0`,
	}
	for _, stmt := range alters {
		if _, err := db.Exec(stmt); err != nil {
			// SQLite does not expose a specific error code for "duplicate column" —
			// the error is a generic SQLITE_ERROR (code 1) with message text.
			// String matching is the standard approach used by SQLite migration tools.
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
		`CREATE INDEX IF NOT EXISTS idx_test_results_status ON test_results(status)`,
		`CREATE INDEX IF NOT EXISTS idx_project_members_project ON project_members(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_project_members_user ON project_members(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_steps_analysis ON agent_steps(analysis_id)`,
		`CREATE INDEX IF NOT EXISTS idx_test_plans_analysis ON test_plans(analysis_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_steps_analysis_step ON agent_steps(analysis_id, step_number)`,
		// Composite indexes for common ORDER BY queries
		`CREATE INDEX IF NOT EXISTS idx_analyses_project_created ON analyses(project_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_test_results_project_created ON test_results(project_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_test_plans_project_created ON test_plans(project_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_test_results_timestamp ON test_results(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_analyses_status ON analyses(status)`,
		`CREATE INDEX IF NOT EXISTS idx_test_plans_status ON test_plans(status)`,
		`CREATE INDEX IF NOT EXISTS idx_analyses_game_url ON analyses(game_url)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_updated ON projects(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_users_created ON users(created_at)`,
	}
	for _, stmt := range indexes {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("creating index: %w", err)
		}
	}

	return nil
}

// collectGameURLs executes a query that returns a single TEXT column and adds each value to dst.
func collectGameURLs(db *sql.DB, query string, dst map[string]bool) {
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("collectGameURLs: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var u string
		if rows.Scan(&u) == nil {
			dst[u] = true
		}
	}
}

// MigrateToProjects auto-creates projects from existing game_url data.
// Idempotent: only runs when unassigned records exist.
func (s *Store) MigrateToProjects() {
	var unassigned int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM analyses WHERE project_id = '' AND game_url != ''`).Scan(&unassigned); err != nil {
		log.Printf("MigrateToProjects: failed to count unassigned analyses: %v", err)
		return
	}
	var unassignedPlans int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM test_plans WHERE project_id = '' AND game_url != ''`).Scan(&unassignedPlans); err != nil {
		log.Printf("MigrateToProjects: failed to count unassigned test plans: %v", err)
		return
	}

	if unassigned == 0 && unassignedPlans == 0 {
		return
	}

	// Collect distinct game_urls
	urls := make(map[string]bool)
	collectGameURLs(s.db, `SELECT DISTINCT game_url FROM analyses WHERE project_id = '' AND game_url != ''`, urls)
	collectGameURLs(s.db, `SELECT DISTINCT game_url FROM test_plans WHERE project_id = '' AND game_url != ''`, urls)

	created := 0
	now := time.Now().Format(time.RFC3339)

	for gameURL := range urls {
		if migrateOneProject(s, gameURL, now, created) {
			created++
		}
	}

	if created > 0 {
		log.Printf("MigrateToProjects: auto-created %d project(s) from existing game_url data", created)
	}
}

// migrateOneProject handles a single project migration inside its own function scope
// so that defer tx.Rollback() fires at the correct time (per-iteration, not at function exit).
func migrateOneProject(s *Store, gameURL, now string, index int) bool {
	name := gameURL
	if parsed, err := neturl.Parse(gameURL); err == nil && parsed.Host != "" {
		name = parsed.Host
	}

	projectID := fmt.Sprintf("proj-%d-%d", time.Now().UnixNano(), index)

	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("MigrateToProjects: failed to begin transaction for %s: %v", gameURL, err)
		return false
	}
	defer tx.Rollback() // no-op after successful Commit

	if _, err = tx.Exec(
		`INSERT INTO projects (id, name, game_url, description, color, icon, tags, settings, created_at, updated_at)
		 VALUES (?, ?, ?, '', '#6366f1', 'gamepad-2', '[]', '{}', ?, ?)`,
		projectID, name, gameURL, now, now,
	); err != nil {
		log.Printf("MigrateToProjects: failed to create project for %s: %v", gameURL, err)
		return false
	}

	if _, err := tx.Exec(`UPDATE analyses SET project_id = ? WHERE game_url = ? AND project_id = ''`, projectID, gameURL); err != nil {
		log.Printf("MigrateToProjects: failed to assign analyses for %s: %v", gameURL, err)
		return false
	}
	if _, err := tx.Exec(`UPDATE test_plans SET project_id = ? WHERE game_url = ? AND project_id = ''`, projectID, gameURL); err != nil {
		log.Printf("MigrateToProjects: failed to assign test plans for %s: %v", gameURL, err)
		return false
	}
	if _, err := tx.Exec(`UPDATE test_results SET project_id = ? WHERE id IN (
		SELECT last_run_id FROM test_plans WHERE project_id = ? AND last_run_id != ''
	)`, projectID, projectID); err != nil {
		log.Printf("MigrateToProjects: failed to assign test results for project %s: %v", projectID, err)
		return false
	}

	if err := tx.Commit(); err != nil {
		log.Printf("MigrateToProjects: failed to commit transaction for %s: %v", gameURL, err)
		return false
	}

	return true
}

// MigrateFromJSON performs a one-time migration of existing JSON data files into SQLite.
func (s *Store) MigrateFromJSON(dataDir string) {
	// Only migrate if DB tables are empty
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM analyses").Scan(&count); err != nil {
		log.Printf("MigrateFromJSON: failed to count analyses: %v", err)
		return
	}
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

	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("Migration: failed to begin transaction for analyses: %v", err)
		return
	}
	defer tx.Rollback() // no-op after successful Commit

	migrated := 0
	for _, a := range file.Analyses {
		resultJSON, marshalErr := marshalToPtr(a.Result)
		if marshalErr != nil {
			log.Printf("Migration: failed to marshal result for analysis %s: %v", a.ID, marshalErr)
			continue
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

		_, err := tx.Exec(
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
	if err := tx.Commit(); err != nil {
		log.Printf("Migration: failed to commit analyses transaction: %v", err)
		return
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

	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("Migration: failed to begin transaction for test results: %v", err)
		return
	}
	defer tx.Rollback() // no-op after successful Commit

	migrated := 0
	for _, r := range file.Results {
		flowsJSON, marshalErr := marshalToPtr(r.Flows)
		if marshalErr != nil {
			log.Printf("Migration: failed to marshal flows for test result %s: %v", r.ID, marshalErr)
			continue
		}
		ts := r.Timestamp
		if ts == "" {
			ts = time.Now().Format(time.RFC3339)
		}

		_, err := tx.Exec(
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
	if err := tx.Commit(); err != nil {
		log.Printf("Migration: failed to commit test results transaction: %v", err)
		return
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

	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("Migration: failed to begin transaction for test plans: %v", err)
		return
	}
	defer tx.Rollback() // no-op after successful Commit

	migrated := 0
	for _, p := range file.Plans {
		flowNamesJSON, marshalErr := marshalToPtr(p.FlowNames)
		if marshalErr != nil {
			log.Printf("Migration: failed to marshal flow names for test plan %s: %v", p.ID, marshalErr)
			continue
		}
		variablesJSON, marshalErr2 := marshalToPtr(p.Variables)
		if marshalErr2 != nil {
			log.Printf("Migration: failed to marshal variables for test plan %s: %v", p.ID, marshalErr2)
			continue
		}
		createdAt := p.CreatedAt
		if createdAt == "" {
			createdAt = time.Now().Format(time.RFC3339)
		}

		_, err := tx.Exec(
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
	if err := tx.Commit(); err != nil {
		log.Printf("Migration: failed to commit test plans transaction: %v", err)
		return
	}
	if migrated > 0 {
		log.Printf("Migration: migrated %d test plans from JSON", migrated)
	}
}
