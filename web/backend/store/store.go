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
	dataDir    string
}

func New(db *sql.DB, flowsDir, reportsDir, configPath string) *Store {
	return &Store{
		db:         db,
		flowsDir:   flowsDir,
		reportsDir: reportsDir,
		configPath: configPath,
	}
}

// SetDataDir sets the data directory path for screenshot storage.
func (s *Store) SetDataDir(dir string) {
	s.dataDir = dir
}

// DataDir returns the data directory path.
func (s *Store) DataDir() string {
	return s.dataDir
}

// Ping checks database connectivity.
func (s *Store) Ping() error {
	return s.db.Ping()
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
	result, err := s.db.Exec(`UPDATE test_plans SET status = ? WHERE status = ?`, StatusFailed, StatusRunning)
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
	result, err := s.db.Exec(`UPDATE analyses SET status = ?, step = '', updated_at = ? WHERE status = ?`, StatusFailed, now, StatusRunning)
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

// marshalToPtr marshals v to a JSON string pointer. Returns nil if v is nil or marshaling fails.
func marshalToPtr(v interface{}) *string {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

// --- Flows ---

func (s *Store) ListFlows() ([]FlowInfo, error) {
	var flows []FlowInfo
	err := filepath.Walk(s.flowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !isYAMLFile(path) {
			return nil
		}
		name, category, relPath := s.flowMeta(path, info)
		if category == "game-mechanics" {
			return nil
		}
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
	resultJSON := marshalToPtr(record.Result)
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

	agentModeInt := 0
	if record.AgentMode {
		agentModeInt = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO analyses (id, game_url, status, step, framework, game_name, flow_count, result, created_by, project_id, modules, agent_mode, profile, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.GameURL, record.Status, record.Step, record.Framework,
		record.GameName, record.FlowCount, resultJSON, createdBy, record.ProjectID, record.Modules, agentModeInt, record.Profile, record.CreatedAt, record.UpdatedAt,
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
	resultJSON := marshalToPtr(result)
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
		`SELECT id, game_url, status, step, framework, game_name, flow_count, result, COALESCE(created_by,''), COALESCE(project_id,''), created_at, updated_at, COALESCE(error_message,''), COALESCE(modules,''), COALESCE(partial_result,''), COALESCE(agent_mode,0), COALESCE(profile,'') FROM analyses WHERE id = ?`, id,
	)
	var a AnalysisRecord
	var resultJSON sql.NullString
	var agentModeInt int
	err := row.Scan(&a.ID, &a.GameURL, &a.Status, &a.Step, &a.Framework, &a.GameName, &a.FlowCount, &resultJSON, &a.CreatedBy, &a.ProjectID, &a.CreatedAt, &a.UpdatedAt, &a.ErrorMessage, &a.Modules, &a.PartialResult, &agentModeInt, &a.Profile)
	if err != nil {
		return nil, fmt.Errorf("analysis not found: %s", id)
	}
	a.AgentMode = agentModeInt != 0
	if resultJSON.Valid && resultJSON.String != "" {
		var parsed interface{}
		if err := json.Unmarshal([]byte(resultJSON.String), &parsed); err == nil {
			a.Result = parsed
		}
	}
	return &a, nil
}

func (s *Store) ListAnalyses(limit, offset int) ([]AnalysisRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, game_url, status, step, framework, game_name, flow_count, COALESCE(created_by,''), COALESCE(project_id,''), COALESCE(modules,''), COALESCE(partial_result,''), created_at, updated_at FROM analyses ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analyses []AnalysisRecord
	for rows.Next() {
		var a AnalysisRecord
		if err := rows.Scan(&a.ID, &a.GameURL, &a.Status, &a.Step, &a.Framework, &a.GameName, &a.FlowCount, &a.CreatedBy, &a.ProjectID, &a.Modules, &a.PartialResult, &a.CreatedAt, &a.UpdatedAt); err != nil {
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
	// Clean up persisted screenshots
	if s.dataDir != "" {
		screenshotDir := filepath.Join(s.dataDir, "screenshots", id)
		if err := os.RemoveAll(screenshotDir); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to clean up screenshots for %s: %v", id, err)
		}
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

// --- Agent Steps ---

func (s *Store) SaveAgentStep(step AgentStepRecord) (int64, error) {
	if step.CreatedAt == "" {
		step.CreatedAt = time.Now().Format(time.RFC3339)
	}
	res, err := s.db.Exec(
		`INSERT INTO agent_steps (analysis_id, step_number, tool_name, input, result, screenshot_path, duration_ms, error, reasoning, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		step.AnalysisID, step.StepNumber, step.ToolName, step.Input, step.Result,
		step.ScreenshotPath, step.DurationMs, step.Error, step.Reasoning, step.CreatedAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateAgentStepScreenshot(id int64, screenshotPath string) error {
	_, err := s.db.Exec(`UPDATE agent_steps SET screenshot_path = ? WHERE id = ?`, screenshotPath, id)
	return err
}

func (s *Store) ListAgentSteps(analysisID string) ([]AgentStepRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, analysis_id, step_number, tool_name, input, result, screenshot_path, duration_ms, error, reasoning, created_at
		 FROM agent_steps WHERE analysis_id = ? ORDER BY step_number ASC`, analysisID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []AgentStepRecord
	for rows.Next() {
		var step AgentStepRecord
		if err := rows.Scan(&step.ID, &step.AnalysisID, &step.StepNumber, &step.ToolName, &step.Input,
			&step.Result, &step.ScreenshotPath, &step.DurationMs, &step.Error, &step.Reasoning, &step.CreatedAt); err != nil {
			continue
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func (s *Store) UpdateAnalysisError(id, errorMessage string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE analyses SET error_message = ?, updated_at = ? WHERE id = ?`, errorMessage, now, id)
	return err
}

func (s *Store) UpdateAnalysisPartialResult(id, partialResult string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE analyses SET partial_result = ?, updated_at = ? WHERE id = ?`, partialResult, now, id)
	return err
}

// --- Test Results (SQLite) ---

func (s *Store) SaveTestResult(result TestResultDetail) error {
	flowsJSON := marshalToPtr(result.Flows)
	ts := result.Timestamp
	if ts == "" {
		ts = time.Now().Format(time.RFC3339)
	}
	var createdBy *string
	if result.CreatedBy != "" {
		createdBy = &result.CreatedBy
	}

	_, err := s.db.Exec(
		`INSERT INTO test_results (id, name, status, timestamp, duration, success_rate, flows, error_output, created_by, project_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.ID, result.Name, result.Status, ts, result.Duration, result.SuccessRate, flowsJSON, result.ErrorOutput, createdBy, result.ProjectID, ts,
	)
	return err
}

func (s *Store) ListTestResults(limit, offset int) ([]TestResultSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, name, status, timestamp, duration, success_rate, COALESCE(project_id,'') FROM test_results ORDER BY timestamp DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TestResultSummary
	for rows.Next() {
		var r TestResultSummary
		if err := rows.Scan(&r.ID, &r.Name, &r.Status, &r.Timestamp, &r.Duration, &r.SuccessRate, &r.ProjectID); err != nil {
			continue
		}
		summaries = append(summaries, r)
	}
	return summaries, rows.Err()
}

func (s *Store) GetTestResult(id string) (*TestResultDetail, error) {
	row := s.db.QueryRow(
		`SELECT id, name, status, timestamp, duration, success_rate, flows, error_output, COALESCE(created_by,''), COALESCE(project_id,'') FROM test_results WHERE id = ?`, id,
	)
	var r TestResultDetail
	var flowsJSON sql.NullString
	err := row.Scan(&r.ID, &r.Name, &r.Status, &r.Timestamp, &r.Duration, &r.SuccessRate, &flowsJSON, &r.ErrorOutput, &r.CreatedBy, &r.ProjectID)
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
	flowNamesJSON := marshalToPtr(plan.FlowNames)
	variablesJSON := marshalToPtr(plan.Variables)
	if plan.CreatedAt == "" {
		plan.CreatedAt = time.Now().Format(time.RFC3339)
	}
	var createdBy *string
	if plan.CreatedBy != "" {
		createdBy = &plan.CreatedBy
	}

	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO test_plans (id, name, description, game_url, flow_names, variables, status, last_run_id, created_by, project_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		plan.ID, plan.Name, plan.Description, plan.GameURL, flowNamesJSON, variablesJSON, plan.Status, plan.LastRunID, createdBy, plan.ProjectID, plan.CreatedAt,
	)
	return err
}

func (s *Store) ListTestPlans(limit, offset int) ([]TestPlanSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, name, status, flow_names, created_at, last_run_id, COALESCE(project_id,'') FROM test_plans ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TestPlanSummary
	for rows.Next() {
		var p TestPlanSummary
		var flowNamesJSON sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &p.Status, &flowNamesJSON, &p.CreatedAt, &p.LastRunID, &p.ProjectID); err != nil {
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
		`SELECT id, name, description, game_url, flow_names, variables, status, last_run_id, COALESCE(created_by,''), COALESCE(project_id,''), created_at FROM test_plans WHERE id = ?`, id,
	)
	var p TestPlan
	var flowNamesJSON, variablesJSON sql.NullString
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.GameURL, &flowNamesJSON, &variablesJSON, &p.Status, &p.LastRunID, &p.CreatedBy, &p.ProjectID, &p.CreatedAt)
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

	if err := s.db.QueryRow(`SELECT COUNT(*), COUNT(CASE WHEN status = 'passed' THEN 1 END), COUNT(CASE WHEN status = 'failed' THEN 1 END), COALESCE(AVG(success_rate), 0) FROM test_results`).Scan(&totalTests, &passedTests, &failedTests, &avgRate); err != nil {
		return nil, fmt.Errorf("querying test stats: %w", err)
	}

	// Recent tests (last 10)
	recent := s.recentTests(10)

	// Build history from test results
	history := s.buildHistoryFromDB(14)

	var totalAnalyses, totalPlans int
	if err := s.db.QueryRow(`SELECT (SELECT COUNT(*) FROM analyses), (SELECT COUNT(*) FROM test_plans)`).Scan(&totalAnalyses, &totalPlans); err != nil {
		return nil, fmt.Errorf("querying entity counts: %w", err)
	}

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

	cutoff := now.AddDate(0, 0, -days).Format(time.RFC3339)
	rows, err := s.db.Query(`SELECT timestamp, status FROM test_results WHERE timestamp >= ?`, cutoff)
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
			if status == StatusPassed {
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
	rows, err := s.db.Query(`SELECT id, email, display_name, role, created_at FROM users ORDER BY created_at LIMIT 500`)
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

// --- Projects ---

func (s *Store) SaveProject(p Project) error {
	tagsJSON, _ := json.Marshal(p.Tags)
	settingsJSON, _ := json.Marshal(p.Settings)
	var createdBy *string
	if p.CreatedBy != "" {
		createdBy = &p.CreatedBy
	}
	_, err := s.db.Exec(
		`INSERT INTO projects (id, name, game_url, description, color, icon, tags, settings, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.GameURL, p.Description, p.Color, p.Icon, string(tagsJSON), string(settingsJSON), createdBy, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (s *Store) GetProject(id string) (*Project, error) {
	row := s.db.QueryRow(
		`SELECT id, name, game_url, description, color, icon, tags, settings, COALESCE(created_by,''), created_at, updated_at FROM projects WHERE id = ?`, id,
	)
	var p Project
	var tagsJSON, settingsJSON string
	err := row.Scan(&p.ID, &p.Name, &p.GameURL, &p.Description, &p.Color, &p.Icon, &tagsJSON, &settingsJSON, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	json.Unmarshal([]byte(tagsJSON), &p.Tags)
	json.Unmarshal([]byte(settingsJSON), &p.Settings)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if p.Settings == nil {
		p.Settings = map[string]string{}
	}
	return &p, nil
}

func (s *Store) ListProjects() ([]ProjectSummary, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.game_url, p.description, p.color, p.icon, p.tags, p.settings,
		       COALESCE(p.created_by,''), p.created_at, p.updated_at,
		       COALESCE(ac.cnt, 0), COALESCE(tp.cnt, 0), COALESCE(tr.cnt, 0), COALESCE(pm.cnt, 0)
		FROM projects p
		LEFT JOIN (SELECT project_id, COUNT(*) AS cnt FROM analyses GROUP BY project_id) ac ON ac.project_id = p.id
		LEFT JOIN (SELECT project_id, COUNT(*) AS cnt FROM test_plans GROUP BY project_id) tp ON tp.project_id = p.id
		LEFT JOIN (SELECT project_id, COUNT(*) AS cnt FROM test_results GROUP BY project_id) tr ON tr.project_id = p.id
		LEFT JOIN (SELECT project_id, COUNT(*) AS cnt FROM project_members GROUP BY project_id) pm ON pm.project_id = p.id
		ORDER BY p.updated_at DESC
		LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []ProjectSummary
	for rows.Next() {
		var ps ProjectSummary
		var tagsJSON, settingsJSON string
		if err := rows.Scan(&ps.ID, &ps.Name, &ps.GameURL, &ps.Description, &ps.Color, &ps.Icon,
			&tagsJSON, &settingsJSON, &ps.CreatedBy, &ps.CreatedAt, &ps.UpdatedAt,
			&ps.AnalysisCount, &ps.PlanCount, &ps.TestCount, &ps.MemberCount); err != nil {
			continue
		}
		json.Unmarshal([]byte(tagsJSON), &ps.Tags)
		json.Unmarshal([]byte(settingsJSON), &ps.Settings)
		if ps.Tags == nil {
			ps.Tags = []string{}
		}
		if ps.Settings == nil {
			ps.Settings = map[string]string{}
		}
		projects = append(projects, ps)
	}
	return projects, rows.Err()
}

func (s *Store) UpdateProject(p Project) error {
	tagsJSON, _ := json.Marshal(p.Tags)
	settingsJSON, _ := json.Marshal(p.Settings)
	result, err := s.db.Exec(
		`UPDATE projects SET name = ?, game_url = ?, description = ?, color = ?, icon = ?, tags = ?, settings = ?, updated_at = ? WHERE id = ?`,
		p.Name, p.GameURL, p.Description, p.Color, p.Icon, string(tagsJSON), string(settingsJSON), p.UpdatedAt, p.ID,
	)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("project not found: %s", p.ID)
	}
	return nil
}

func (s *Store) DeleteProject(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Unassign entities before deleting (CASCADE handles project_members)
	if _, err := tx.Exec(`UPDATE analyses SET project_id = '' WHERE project_id = ?`, id); err != nil {
		return fmt.Errorf("unassign analyses: %w", err)
	}
	if _, err := tx.Exec(`UPDATE test_plans SET project_id = '' WHERE project_id = ?`, id); err != nil {
		return fmt.Errorf("unassign test plans: %w", err)
	}
	if _, err := tx.Exec(`UPDATE test_results SET project_id = '' WHERE project_id = ?`, id); err != nil {
		return fmt.Errorf("unassign test results: %w", err)
	}

	result, err := tx.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	return tx.Commit()
}

// GetProjectMemberRole returns the role of a user within a project, or an error if not a member.
func (s *Store) GetProjectMemberRole(projectID, userID string) (string, error) {
	var role string
	err := s.db.QueryRow(`SELECT role FROM project_members WHERE project_id = ? AND user_id = ?`, projectID, userID).Scan(&role)
	if err != nil {
		return "", fmt.Errorf("not a project member")
	}
	return role, nil
}

// --- Project Members ---

func (s *Store) AddProjectMember(m ProjectMember) error {
	_, err := s.db.Exec(
		`INSERT INTO project_members (id, project_id, user_id, role, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		m.ID, m.ProjectID, m.UserID, m.Role, m.CreatedAt,
	)
	return err
}

func (s *Store) RemoveProjectMember(projectID, userID string) error {
	result, err := s.db.Exec(`DELETE FROM project_members WHERE project_id = ? AND user_id = ?`, projectID, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

func (s *Store) ListProjectMembers(projectID string) ([]ProjectMember, error) {
	rows, err := s.db.Query(`
		SELECT pm.id, pm.project_id, pm.user_id, pm.role, pm.created_at,
		       u.email, u.display_name
		FROM project_members pm
		JOIN users u ON u.id = pm.user_id
		WHERE pm.project_id = ?
		ORDER BY pm.created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []ProjectMember
	for rows.Next() {
		var m ProjectMember
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.UserID, &m.Role, &m.CreatedAt, &m.Email, &m.DisplayName); err != nil {
			continue
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *Store) UpdateProjectMemberRole(projectID, userID, role string) error {
	result, err := s.db.Exec(`UPDATE project_members SET role = ? WHERE project_id = ? AND user_id = ?`, role, projectID, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

// --- Project-scoped queries ---

func (s *Store) ListAnalysesByProject(projectID string) ([]AnalysisRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, game_url, status, step, framework, game_name, flow_count, COALESCE(created_by,''), COALESCE(project_id,''), COALESCE(modules,''), COALESCE(partial_result,''), created_at, updated_at FROM analyses WHERE project_id = ? ORDER BY created_at DESC LIMIT 200`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analyses []AnalysisRecord
	for rows.Next() {
		var a AnalysisRecord
		if err := rows.Scan(&a.ID, &a.GameURL, &a.Status, &a.Step, &a.Framework, &a.GameName, &a.FlowCount, &a.CreatedBy, &a.ProjectID, &a.Modules, &a.PartialResult, &a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		analyses = append(analyses, a)
	}
	return analyses, rows.Err()
}

func (s *Store) ListTestPlansByProject(projectID string) ([]TestPlanSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, name, status, flow_names, created_at, last_run_id, COALESCE(project_id,'') FROM test_plans WHERE project_id = ? ORDER BY created_at DESC LIMIT 200`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TestPlanSummary
	for rows.Next() {
		var p TestPlanSummary
		var flowNamesJSON sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &p.Status, &flowNamesJSON, &p.CreatedAt, &p.LastRunID, &p.ProjectID); err != nil {
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

func (s *Store) ListTestResultsByProject(projectID string) ([]TestResultSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, name, status, timestamp, duration, success_rate, COALESCE(project_id,'') FROM test_results WHERE project_id = ? ORDER BY timestamp DESC LIMIT 200`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TestResultSummary
	for rows.Next() {
		var r TestResultSummary
		if err := rows.Scan(&r.ID, &r.Name, &r.Status, &r.Timestamp, &r.Duration, &r.SuccessRate, &r.ProjectID); err != nil {
			continue
		}
		summaries = append(summaries, r)
	}
	return summaries, rows.Err()
}

func (s *Store) GetStatsByProject(projectID string) (*Stats, error) {
	var totalTests, passedTests, failedTests int
	var avgRate float64

	if err := s.db.QueryRow(`SELECT COUNT(*), COUNT(CASE WHEN status = 'passed' THEN 1 END), COUNT(CASE WHEN status = 'failed' THEN 1 END), COALESCE(AVG(success_rate), 0) FROM test_results WHERE project_id = ?`, projectID).Scan(&totalTests, &passedTests, &failedTests, &avgRate); err != nil {
		return nil, fmt.Errorf("querying project test stats: %w", err)
	}

	recent := s.recentTestsByProject(projectID, 10)
	history := s.buildHistoryFromDB(14) // reuse global for now

	var totalAnalyses, totalPlans int
	if err := s.db.QueryRow(`SELECT (SELECT COUNT(*) FROM analyses WHERE project_id = ?), (SELECT COUNT(*) FROM test_plans WHERE project_id = ?)`, projectID, projectID).Scan(&totalAnalyses, &totalPlans); err != nil {
		return nil, fmt.Errorf("querying project entity counts: %w", err)
	}

	return &Stats{
		TotalTests:     totalTests,
		PassedTests:    passedTests,
		FailedTests:    failedTests,
		AvgDuration:    "42s",
		AvgSuccessRate: float64(int(avgRate*10)) / 10,
		TotalAnalyses:  totalAnalyses,
		TotalFlows:     0,
		TotalPlans:     totalPlans,
		RecentTests:    recent,
		History:        history,
	}, nil
}

func (s *Store) recentTestsByProject(projectID string, limit int) []TestResultSummary {
	rows, err := s.db.Query(
		`SELECT id, name, status, timestamp, duration, success_rate FROM test_results WHERE project_id = ? ORDER BY timestamp DESC LIMIT ?`, projectID, limit,
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
