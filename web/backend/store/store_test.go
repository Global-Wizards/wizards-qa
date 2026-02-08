package store

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, *Store) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening in-memory db: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=OFF"); err != nil {
		t.Fatalf("setting pragma: %v", err)
	}
	if err := createTables(db); err != nil {
		t.Fatalf("creating tables: %v", err)
	}
	if err := runMigrations(db); err != nil {
		t.Fatalf("running migrations: %v", err)
	}
	s := New(db, t.TempDir(), t.TempDir(), "")
	return db, s
}

func TestPing(t *testing.T) {
	_, s := setupTestDB(t)
	if err := s.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestCreateAndGetUser(t *testing.T) {
	db, s := setupTestDB(t)
	now := time.Now().Format(time.RFC3339)

	_, err := db.Exec(
		`INSERT INTO users (id, email, display_name, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"u1", "test@example.com", "Test User", "hash123", "member", now, now,
	)
	if err != nil {
		t.Fatalf("inserting user: %v", err)
	}

	user, err := s.GetUserByEmail("test@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if user.ID != "u1" {
		t.Errorf("user ID = %q, want %q", user.ID, "u1")
	}
	if user.Email != "test@example.com" {
		t.Errorf("email = %q, want %q", user.Email, "test@example.com")
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	_, s := setupTestDB(t)
	_, err := s.GetUserByEmail("nobody@example.com")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestGetProjectMemberRole(t *testing.T) {
	db, s := setupTestDB(t)
	now := time.Now().Format(time.RFC3339)

	// Create user and project
	db.Exec(`INSERT INTO users (id, email, display_name, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"u1", "test@example.com", "Test User", "hash123", "member", now, now)
	db.Exec(`INSERT INTO projects (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"p1", "Test Project", now, now)
	db.Exec(`INSERT INTO project_members (id, project_id, user_id, role, created_at) VALUES (?, ?, ?, ?, ?)`,
		"pm1", "p1", "u1", "owner", now)

	role, err := s.GetProjectMemberRole("p1", "u1")
	if err != nil {
		t.Fatalf("GetProjectMemberRole failed: %v", err)
	}
	if role != "owner" {
		t.Errorf("role = %q, want %q", role, "owner")
	}
}

func TestGetProjectMemberRole_NotMember(t *testing.T) {
	_, s := setupTestDB(t)
	_, err := s.GetProjectMemberRole("p1", "nonexistent-user")
	if err == nil {
		t.Fatal("expected error for non-member")
	}
}

func TestListAnalysesPagination(t *testing.T) {
	db, s := setupTestDB(t)
	now := time.Now().Format(time.RFC3339)

	for i := 0; i < 10; i++ {
		id := "a" + string(rune('0'+i))
		db.Exec(`INSERT INTO analyses (id, game_url, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			id, "https://example.com", "completed", now, now)
	}

	// Get first page
	results, err := s.ListAnalyses(3, 0)
	if err != nil {
		t.Fatalf("ListAnalyses failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Get second page
	results2, err := s.ListAnalyses(3, 3)
	if err != nil {
		t.Fatalf("ListAnalyses (page 2) failed: %v", err)
	}
	if len(results2) != 3 {
		t.Errorf("expected 3 results on page 2, got %d", len(results2))
	}
}

func TestListTestResultsPagination(t *testing.T) {
	db, s := setupTestDB(t)
	now := time.Now().Format(time.RFC3339)

	for i := 0; i < 5; i++ {
		id := "t" + string(rune('0'+i))
		db.Exec(`INSERT INTO test_results (id, name, status, timestamp, created_at) VALUES (?, ?, ?, ?, ?)`,
			id, "Test "+id, "passed", now, now)
	}

	results, err := s.ListTestResults(2, 0)
	if err != nil {
		t.Fatalf("ListTestResults failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestListTestPlansPagination(t *testing.T) {
	db, s := setupTestDB(t)
	now := time.Now().Format(time.RFC3339)

	for i := 0; i < 5; i++ {
		id := "tp" + string(rune('0'+i))
		db.Exec(`INSERT INTO test_plans (id, name, status, created_at) VALUES (?, ?, ?, ?)`,
			id, "Plan "+id, "draft", now)
	}

	results, err := s.ListTestPlans(2, 0)
	if err != nil {
		t.Fatalf("ListTestPlans failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestListProjects(t *testing.T) {
	db, s := setupTestDB(t)
	now := time.Now().Format(time.RFC3339)

	db.Exec(`INSERT INTO projects (id, name, game_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"p1", "Project 1", "https://example.com", now, now)
	db.Exec(`INSERT INTO projects (id, name, game_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"p2", "Project 2", "https://example2.com", now, now)

	projects, err := s.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestRecoverOrphanedRuns(t *testing.T) {
	db, s := setupTestDB(t)
	now := time.Now().Format(time.RFC3339)

	db.Exec(`INSERT INTO test_plans (id, name, status, created_at) VALUES (?, ?, ?, ?)`,
		"tp1", "Running plan", "running", now)
	db.Exec(`INSERT INTO test_plans (id, name, status, created_at) VALUES (?, ?, ?, ?)`,
		"tp2", "Draft plan", "draft", now)

	s.RecoverOrphanedRuns()

	var status string
	db.QueryRow(`SELECT status FROM test_plans WHERE id = 'tp1'`).Scan(&status)
	if status != "failed" {
		t.Errorf("orphaned plan status = %q, want %q", status, "failed")
	}
	db.QueryRow(`SELECT status FROM test_plans WHERE id = 'tp2'`).Scan(&status)
	if status != "draft" {
		t.Errorf("draft plan status = %q, want %q", status, "draft")
	}
}

func TestRecoverOrphanedAnalyses(t *testing.T) {
	db, s := setupTestDB(t)
	now := time.Now().Format(time.RFC3339)

	db.Exec(`INSERT INTO analyses (id, game_url, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"a1", "https://example.com", "running", now, now)

	s.RecoverOrphanedAnalyses()

	var status string
	db.QueryRow(`SELECT status FROM analyses WHERE id = 'a1'`).Scan(&status)
	if status != "failed" {
		t.Errorf("orphaned analysis status = %q, want %q", status, "failed")
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"report.md", "markdown"},
		{"report.markdown", "markdown"},
		{"data.json", "json"},
		{"results.xml", "junit"},
		{"page.html", "html"},
		{"notes.txt", "text"},
		{"file.xyz", "unknown"},
	}
	for _, tt := range tests {
		got := detectFormat(tt.name)
		if got != tt.want {
			t.Errorf("detectFormat(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestIsYAMLFile(t *testing.T) {
	if !isYAMLFile("flow.yaml") {
		t.Error("expected .yaml to be YAML")
	}
	if !isYAMLFile("flow.yml") {
		t.Error("expected .yml to be YAML")
	}
	if isYAMLFile("flow.json") {
		t.Error("expected .json to not be YAML")
	}
}
