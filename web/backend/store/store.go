package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	varPattern    = regexp.MustCompile(`\{\{(\w+)\}\}`)
	safeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
)

type Store struct {
	db         *sql.DB
	flowsDir   string
	reportsDir string
	configPath string
}

func New(db *sql.DB, flowsDir, reportsDir, configPath string) *Store {
	return &Store{
		db:         db,
		flowsDir:   flowsDir,
		reportsDir: reportsDir,
		configPath: configPath,
	}
}

// ValidateDirectories checks that required directories exist.
func (s *Store) ValidateDirectories() error {
	info, err := os.Stat(s.flowsDir)
	if err != nil {
		return fmt.Errorf("flows directory not found: %s: %w", s.flowsDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("flows path is not a directory: %s", s.flowsDir)
	}
	return nil
}

// RecoverOrphanedRuns marks any "running" test plans as failed (crash recovery).
func (s *Store) RecoverOrphanedRuns() {
	result, err := s.db.Exec(`UPDATE test_plans SET status = 'failed' WHERE status = 'running'`)
	if err != nil {
		log.Printf("Warning: failed to recover orphaned test plans: %v", err)
		return
	}
	if n, _ := result.RowsAffected(); n > 0 {
		log.Printf("Recovered %d orphaned running test plans", n)
	}
}

// RecoverOrphanedAnalyses marks any "running" analyses as "failed" (crash recovery).
func (s *Store) RecoverOrphanedAnalyses() {
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.Exec(`UPDATE analyses SET status = 'failed', step = '', updated_at = ? WHERE status = 'running'`, now)
	if err != nil {
		log.Printf("Warning: failed to recover orphaned analyses: %v", err)
		return
	}
	if n, _ := result.RowsAffected(); n > 0 {
		log.Printf("Recovered %d orphaned running analyses", n)
	}
}

// --- DRY helpers ---

func detectFormat(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".md", ".markdown":
		return "markdown"
	case ".json":
		return "json"
	case ".xml":
		return "junit"
	case ".html":
		return "html"
	case ".txt":
		return "text"
	default:
		return "unknown"
	}
}

func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func (s *Store) flowMeta(path string, info os.FileInfo) (name, category, relPath string) {
	ext := strings.ToLower(filepath.Ext(path))
	name = strings.TrimSuffix(info.Name(), ext)
	category = "general"
	relDir := filepath.Dir(path)
	if relDir != s.flowsDir {
		category = filepath.Base(relDir)
	}
	relPath, err := filepath.Rel(filepath.Dir(s.flowsDir), path)
	if err != nil || relPath == "" {
		relPath = path
	}
	return
}

func isSafeName(name string) bool {
	return safeNameRegex.MatchString(name)
}

// --- Flows ---

func (s *Store) ListFlows() ([]FlowInfo, error) {
	var flows []FlowInfo
	err := filepath.Walk(s.flowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !isYAMLFile(path) {
			return nil
		}
		name, category, relPath := s.flowMeta(path, info)
		flows = append(flows, FlowInfo{Name: name, Category: category, Path: relPath})
		return nil
	})
	return flows, err
}

func (s *Store) GetFlow(name string) (*FlowDetail, error) {
	if !isSafeName(name) {
		return nil, fmt.Errorf("invalid flow name: %s", name)
	}
	flows, err := s.ListFlows()
	if err != nil {
		return nil, err
	}
	for _, f := range flows {
		if f.Name == name {
			fullPath := filepath.Join(filepath.Dir(s.flowsDir), f.Path)
			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				return nil, fmt.Errorf("resolving path: %w", err)
			}
			absBase, _ := filepath.Abs(filepath.Dir(s.flowsDir))
			if !strings.HasPrefix(absPath, absBase) {
				return nil, fmt.Errorf("path traversal detected")
			}
			content, err := os.ReadFile(absPath)
			if err != nil {
				return nil, fmt.Errorf("reading flow file: %w", err)
			}
			return &FlowDetail{
				Name:     f.Name,
				Category: f.Category,
				Path:     f.Path,
				Content:  string(content),
			}, nil
		}
	}
	return nil, fmt.Errorf("flow not found: %s", name)
}

// --- Reports ---

func (s *Store) ListReports() ([]ReportInfo, error) {
	var reports []ReportInfo
	entries, err := os.ReadDir(s.reportsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return reports, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		reports = append(reports, ReportInfo{
			ID:        entry.Name(),
			Name:      strings.TrimSuffix(entry.Name(), ext),
			Format:    detectFormat(entry.Name()),
			Timestamp: info.ModTime().Format(time.RFC3339),
			Size:      formatSize(info.Size()),
		})
	}
	return reports, nil
}

func (s *Store) GetReport(id string) (*ReportDetail, error) {
	id = filepath.Base(id)
	if id == "." || id == ".." || id == "" {
		return nil, fmt.Errorf("invalid report ID")
	}
	path := filepath.Join(s.reportsDir, id)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading report: %w", err)
	}
	ext := strings.ToLower(filepath.Ext(id))
	return &ReportDetail{
		ID:      id,
		Name:    strings.TrimSuffix(id, ext),
		Format:  detectFormat(id),
		Content: string(content),
	}, nil
}

// --- Analyses (SQLite) ---

func (s *Store) SaveAnalysis(record AnalysisRecord) error {
	var resultJSON *string
	if record.Result != nil {
		b, err := json.Marshal(record.Result)
		if err == nil {
			str := string(b)
			resultJSON = &str
		}
	}
	now := time.Now().Format(time.RFC3339)
	if record.CreatedAt == "" {
		record.CreatedAt = now
	}
	if record.UpdatedAt == "" {
		record.UpdatedAt = now
	}
	var createdBy *string
	if record.CreatedBy != "" {
		createdBy = &record.CreatedBy
	}

	_, err := s.db.Exec(
		`INSERT INTO analyses (id, game_url, status, step, framework, game_name, flow_count, result, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.GameURL, record.Status, record.Step, record.Framework,
		record.GameName, record.FlowCount, resultJSON, createdBy, record.CreatedAt, record.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateAnalysisStatus(id, status, step string) error {
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.Exec(
		`UPDATE analyses SET status = ?, step = ?, updated_at = ? WHERE id = ?`,
		status, step, now, id,
	)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("analysis not found: %s", id)
	}
	return nil
}

func (s *Store) UpdateAnalysisResult(id, status string, result interface{}, gameName, framework string, flowCount int) error {
	now := time.Now().Format(time.RFC3339)
	var resultJSON *string
	if result != nil {
		b, err := json.Marshal(result)
		if err == nil {
			str := string(b)
			resultJSON = &str
		}
	}
	res, err := s.db.Exec(
		`UPDATE analyses SET status = ?, step = '', result = ?, game_name = ?, framework = ?, flow_count = ?, updated_at = ? WHERE id = ?`,
		status, resultJSON, gameName, framework, flowCount, now, id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("analysis not found: %s", id)
	}
	return nil
}

func (s *Store) GetAnalysis(id string) (*AnalysisRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, game_url, status, step, framework, game_name, flow_count, result, COALESCE(created_by,''), created_at, updated_at FROM analyses WHERE id = ?`, id,
	)
	var a AnalysisRecord
	var resultJSON sql.NullString
	err := row.Scan(&a.ID, &a.GameURL, &a.Status, &a.Step, &a.Framework, &a.GameName, &a.FlowCount, &resultJSON, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("analysis not found: %s", id)
	}
	if resultJSON.Valid && resultJSON.String != "" {
		var parsed interface{}
		if err := json.Unmarshal([]byte(resultJSON.String), &parsed); err == nil {
			a.Result = parsed
		}
	}
	return &a, nil
}

func (s *Store) ListAnalyses() ([]AnalysisRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, game_url, status, step, framework, game_name, flow_count, COALESCE(created_by,''), created_at, updated_at FROM analyses ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analyses []AnalysisRecord
	for rows.Next() {
		var a AnalysisRecord
		if err := rows.Scan(&a.ID, &a.GameURL, &a.Status, &a.Step, &a.Framework, &a.GameName, &a.FlowCount, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		analyses = append(analyses, a)
	}
	return analyses, rows.Err()
}

func (s *Store) DeleteAnalysis(id string) error {
	result, err := s.db.Exec(`DELETE FROM analyses WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("analysis not found: %s", id)
	}
	// Clean up generated flow files
	generatedDir := filepath.Join(s.flowsDir, "generated", id)
	if err := os.RemoveAll(generatedDir); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to clean up generated flows for %s: %v", id, err)
	}
	return nil
}

func (s *Store) CountAnalyses() int {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM analyses").Scan(&count); err != nil {
		log.Printf("Warning: failed to count analyses: %v", err)
	}
	return count
}

// --- Test Results (SQLite) ---

func (s *Store) SaveTestResult(result TestResultDetail) error {
	var flowsJSON *string
	if result.Flows != nil {
		b, err := json.Marshal(result.Flows)
		if err == nil {
			str := string(b)
			flowsJSON = &str
		}
	}
	ts := result.Timestamp
	if ts == "" {
		ts = time.Now().Format(time.RFC3339)
	}
	var createdBy *string
	if result.CreatedBy != "" {
		createdBy = &result.CreatedBy
	}

	_, err := s.db.Exec(
		`INSERT INTO test_results (id, name, status, timestamp, duration, success_rate, flows, error_output, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.ID, result.Name, result.Status, ts, result.Duration, result.SuccessRate, flowsJSON, result.ErrorOutput, createdBy, ts,
	)
	return err
}

func (s *Store) ListTestResults() ([]TestResultSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, name, status, timestamp, duration, success_rate FROM test_results ORDER BY timestamp DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TestResultSummary
	for rows.Next() {
		var r TestResultSummary
		if err := rows.Scan(&r.ID, &r.Name, &r.Status, &r.Timestamp, &r.Duration, &r.SuccessRate); err != nil {
			continue
		}
		summaries = append(summaries, r)
	}
	return summaries, rows.Err()
}

func (s *Store) GetTestResult(id string) (*TestResultDetail, error) {
	row := s.db.QueryRow(
		`SELECT id, name, status, timestamp, duration, success_rate, flows, error_output, COALESCE(created_by,'') FROM test_results WHERE id = ?`, id,
	)
	var r TestResultDetail
	var flowsJSON sql.NullString
	err := row.Scan(&r.ID, &r.Name, &r.Status, &r.Timestamp, &r.Duration, &r.SuccessRate, &flowsJSON, &r.ErrorOutput, &r.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("test result not found: %s", id)
	}
	if flowsJSON.Valid && flowsJSON.String != "" {
		if err := json.Unmarshal([]byte(flowsJSON.String), &r.Flows); err != nil {
			log.Printf("Warning: failed to unmarshal flows JSON for test %s: %v", id, err)
		}
	}
	return &r, nil
}

// --- Test Plans (SQLite) ---

func (s *Store) SaveTestPlan(plan TestPlan) error {
	var flowNamesJSON *string
	if plan.FlowNames != nil {
		b, _ := json.Marshal(plan.FlowNames)
		str := string(b)
		flowNamesJSON = &str
	}
	var variablesJSON *string
	if plan.Variables != nil {
		b, _ := json.Marshal(plan.Variables)
		str := string(b)
		variablesJSON = &str
	}
	if plan.CreatedAt == "" {
		plan.CreatedAt = time.Now().Format(time.RFC3339)
	}
	var createdBy *string
	if plan.CreatedBy != "" {
		createdBy = &plan.CreatedBy
	}

	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO test_plans (id, name, description, game_url, flow_names, variables, status, last_run_id, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		plan.ID, plan.Name, plan.Description, plan.GameURL, flowNamesJSON, variablesJSON, plan.Status, plan.LastRunID, createdBy, plan.CreatedAt,
	)
	return err
}

func (s *Store) ListTestPlans() ([]TestPlanSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, name, status, flow_names, created_at, last_run_id FROM test_plans ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TestPlanSummary
	for rows.Next() {
		var p TestPlanSummary
		var flowNamesJSON sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &p.Status, &flowNamesJSON, &p.CreatedAt, &p.LastRunID); err != nil {
			continue
		}
		if flowNamesJSON.Valid && flowNamesJSON.String != "" {
			var names []string
			if err := json.Unmarshal([]byte(flowNamesJSON.String), &names); err != nil {
				log.Printf("Warning: failed to unmarshal flow names for plan %s: %v", p.ID, err)
			}
			p.FlowCount = len(names)
		}
		summaries = append(summaries, p)
	}
	return summaries, rows.Err()
}

func (s *Store) GetTestPlan(id string) (*TestPlan, error) {
	row := s.db.QueryRow(
		`SELECT id, name, description, game_url, flow_names, variables, status, last_run_id, COALESCE(created_by,''), created_at FROM test_plans WHERE id = ?`, id,
	)
	var p TestPlan
	var flowNamesJSON, variablesJSON sql.NullString
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.GameURL, &flowNamesJSON, &variablesJSON, &p.Status, &p.LastRunID, &p.CreatedBy, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("test plan not found: %s", id)
	}
	if flowNamesJSON.Valid && flowNamesJSON.String != "" {
		if err := json.Unmarshal([]byte(flowNamesJSON.String), &p.FlowNames); err != nil {
			log.Printf("Warning: failed to unmarshal flow names for plan %s: %v", id, err)
		}
	}
	if variablesJSON.Valid && variablesJSON.String != "" {
		if err := json.Unmarshal([]byte(variablesJSON.String), &p.Variables); err != nil {
			log.Printf("Warning: failed to unmarshal variables for plan %s: %v", id, err)
		}
	}
	return &p, nil
}

func (s *Store) UpdateTestPlanStatus(id, status, lastRunID string) error {
	query := `UPDATE test_plans SET status = ?`
	args := []interface{}{status}
	if lastRunID != "" {
		query += `, last_run_id = ?`
		args = append(args, lastRunID)
	}
	query += ` WHERE id = ?`
	args = append(args, id)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("test plan not found: %s", id)
	}
	return nil
}

func (s *Store) DeleteTestPlan(id string) error {
	result, err := s.db.Exec(`DELETE FROM test_plans WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("test plan not found: %s", id)
	}
	return nil
}

// --- Stats (SQLite aggregations) ---

func (s *Store) GetStats() (*Stats, error) {
	var totalTests, passedTests, failedTests int
	var avgRate float64

	s.db.QueryRow(`SELECT COUNT(*) FROM test_results`).Scan(&totalTests)
	s.db.QueryRow(`SELECT COUNT(*) FROM test_results WHERE status = 'passed'`).Scan(&passedTests)
	s.db.QueryRow(`SELECT COUNT(*) FROM test_results WHERE status = 'failed'`).Scan(&failedTests)
	s.db.QueryRow(`SELECT COALESCE(AVG(success_rate), 0) FROM test_results`).Scan(&avgRate)

	// Recent tests (last 10)
	recent := s.recentTests(10)

	// Build history from test results
	history := s.buildHistoryFromDB(14)

	var totalAnalyses, totalPlans int
	s.db.QueryRow(`SELECT COUNT(*) FROM analyses`).Scan(&totalAnalyses)
	s.db.QueryRow(`SELECT COUNT(*) FROM test_plans`).Scan(&totalPlans)

	flowCount := 0
	if flows, err := s.ListFlows(); err == nil {
		flowCount = len(flows)
	}

	return &Stats{
		TotalTests:     totalTests,
		PassedTests:    passedTests,
		FailedTests:    failedTests,
		AvgDuration:    "42s",
		AvgSuccessRate: float64(int(avgRate*10)) / 10,
		TotalAnalyses:  totalAnalyses,
		TotalFlows:     flowCount,
		TotalPlans:     totalPlans,
		RecentTests:    recent,
		History:        history,
	}, nil
}

func (s *Store) recentTests(limit int) []TestResultSummary {
	rows, err := s.db.Query(
		`SELECT id, name, status, timestamp, duration, success_rate FROM test_results ORDER BY timestamp DESC LIMIT ?`, limit,
	)
	if err != nil {
		return []TestResultSummary{}
	}
	defer rows.Close()

	var results []TestResultSummary
	for rows.Next() {
		var r TestResultSummary
		if err := rows.Scan(&r.ID, &r.Name, &r.Status, &r.Timestamp, &r.Duration, &r.SuccessRate); err != nil {
			continue
		}
		results = append(results, r)
	}
	if results == nil {
		return []TestResultSummary{}
	}
	return results
}

func (s *Store) buildHistoryFromDB(days int) []HistoryPoint {
	now := time.Now()
	history := make([]HistoryPoint, 0, days)

	rows, err := s.db.Query(`SELECT timestamp, status FROM test_results`)
	if err != nil {
		// Return empty history points
		for i := days - 1; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			history = append(history, HistoryPoint{Date: d.Format("Jan 02")})
		}
		return history
	}
	defer rows.Close()

	dayMap := make(map[string]*HistoryPoint)
	for i := 0; i < days; i++ {
		d := now.AddDate(0, 0, -i)
		key := d.Format("Jan 02")
		dayMap[key] = &HistoryPoint{Date: key}
	}

	for rows.Next() {
		var ts, status string
		if err := rows.Scan(&ts, &status); err != nil {
			continue
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			continue
		}
		key := t.Format("Jan 02")
		if pt, ok := dayMap[key]; ok {
			if status == "passed" {
				pt.Passed++
			} else {
				pt.Failed++
			}
		}
	}

	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		key := d.Format("Jan 02")
		if pt, ok := dayMap[key]; ok {
			history = append(history, *pt)
		}
	}
	return history
}

// --- Config ---

func (s *Store) GetConfig() (*ConfigData, error) {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ConfigData{}, nil
		}
		return nil, err
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	config := &ConfigData{
		ExtraConfig: make(map[string]string),
	}

	for key, val := range raw {
		strVal := fmt.Sprintf("%v", val)
		lowKey := strings.ToLower(key)

		if strings.Contains(lowKey, "key") || strings.Contains(lowKey, "token") || strings.Contains(lowKey, "secret") || strings.Contains(lowKey, "password") {
			strVal = "***REDACTED***"
		}

		switch lowKey {
		case "gameurl", "game_url":
			config.GameURL = strVal
		case "aiprovider", "ai_provider":
			config.AIProvider = strVal
		case "aimodel", "ai_model":
			config.AIModel = strVal
		case "outputdir", "output_dir":
			config.OutputDir = strVal
		case "timeout":
			config.Timeout = 30
		default:
			config.ExtraConfig[key] = strVal
		}
	}

	return config, nil
}

// --- Templates ---

func (s *Store) ListTemplates() ([]TemplateInfo, error) {
	var templates []TemplateInfo
	err := filepath.Walk(s.flowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !isYAMLFile(path) {
			return nil
		}
		name, category, relPath := s.flowMeta(path, info)
		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Warning: could not read template %s: %v", path, err)
			return nil
		}
		matches := varPattern.FindAllStringSubmatch(string(content), -1)
		seen := make(map[string]bool)
		var variables []string
		for _, m := range matches {
			if !seen[m[1]] {
				seen[m[1]] = true
				variables = append(variables, m[1])
			}
		}
		templates = append(templates, TemplateInfo{
			Name:      name,
			Category:  category,
			Path:      relPath,
			Variables: variables,
		})
		return nil
	})
	return templates, err
}

func (s *Store) FlowsDir() string {
	return s.flowsDir
}

func (s *Store) SaveGeneratedFlows(analysisID string, srcDir string) error {
	dstDir := filepath.Join(s.flowsDir, "generated", analysisID)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("creating generated flows dir: %w", err)
	}
	filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !isYAMLFile(path) {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			log.Printf("Warning: could not read generated flow %s: %v", info.Name(), readErr)
			return nil
		}
		if writeErr := os.WriteFile(filepath.Join(dstDir, info.Name()), content, 0644); writeErr != nil {
			log.Printf("Warning: could not write generated flow %s: %v", info.Name(), writeErr)
		}
		return nil
	})
	return nil
}

// --- User methods ---

func (s *Store) CreateUser(user User) error {
	_, err := s.db.Exec(
		`INSERT INTO users (id, email, display_name, password_hash, role, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.DisplayName, user.PasswordHash, user.Role, user.CreatedAt, user.CreatedAt,
	)
	return err
}

func (s *Store) GetUserByEmail(email string) (*User, error) {
	row := s.db.QueryRow(
		`SELECT id, email, display_name, password_hash, role, created_at FROM users WHERE email = ?`, email,
	)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return &u, nil
}

func (s *Store) GetUserByID(id string) (*User, error) {
	row := s.db.QueryRow(
		`SELECT id, email, display_name, password_hash, role, created_at FROM users WHERE id = ?`, id,
	)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return &u, nil
}

func (s *Store) ListUsers() ([]UserSummary, error) {
	rows, err := s.db.Query(`SELECT id, email, display_name, role, created_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserSummary
	for rows.Next() {
		var u UserSummary
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) UpdateUserRole(id, role string) error {
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.Exec(`UPDATE users SET role = ?, updated_at = ? WHERE id = ?`, role, now, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

func (s *Store) UserCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
