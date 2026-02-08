package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

	return db, nil
}

func createTables(db *sql.DB) error {
	stmts := []string{
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
