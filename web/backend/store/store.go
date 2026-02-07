package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var varPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

type Store struct {
	flowsDir   string
	reportsDir string
	dataDir    string
	configPath string
	mu         sync.RWMutex
}

func New(flowsDir, reportsDir, dataDir, configPath string) *Store {
	return &Store{
		flowsDir:   flowsDir,
		reportsDir: reportsDir,
		dataDir:    dataDir,
		configPath: configPath,
	}
}

// ListFlows walks the flows directory for .yaml files.
func (s *Store) ListFlows() ([]FlowInfo, error) {
	var flows []FlowInfo

	err := filepath.Walk(s.flowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		name := strings.TrimSuffix(info.Name(), ext)
		category := "general"
		relDir := filepath.Dir(path)
		if relDir != s.flowsDir {
			category = filepath.Base(relDir)
		}
		relPath, _ := filepath.Rel(filepath.Dir(s.flowsDir), path)
		if relPath == "" {
			relPath = path
		}

		flows = append(flows, FlowInfo{
			Name:     name,
			Category: category,
			Path:     relPath,
		})
		return nil
	})

	return flows, err
}

// GetFlow reads a specific flow file by name.
func (s *Store) GetFlow(name string) (*FlowDetail, error) {
	flows, err := s.ListFlows()
	if err != nil {
		return nil, err
	}

	for _, f := range flows {
		if f.Name == name {
			fullPath := filepath.Join(filepath.Dir(s.flowsDir), f.Path)
			content, err := os.ReadFile(fullPath)
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

// ListReports scans the reports directory.
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
		format := "unknown"
		switch ext {
		case ".md", ".markdown":
			format = "markdown"
		case ".json":
			format = "json"
		case ".xml":
			format = "junit"
		case ".html":
			format = "html"
		case ".txt":
			format = "text"
		}

		name := strings.TrimSuffix(entry.Name(), ext)
		size := formatSize(info.Size())

		reports = append(reports, ReportInfo{
			ID:        entry.Name(),
			Name:      name,
			Format:    format,
			Timestamp: info.ModTime().Format(time.RFC3339),
			Size:      size,
		})
	}

	return reports, nil
}

// GetReport reads a report file.
func (s *Store) GetReport(id string) (*ReportDetail, error) {
	path := filepath.Join(s.reportsDir, id)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading report: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(id))
	format := "text"
	switch ext {
	case ".md", ".markdown":
		format = "markdown"
	case ".json":
		format = "json"
	case ".xml":
		format = "junit"
	}

	return &ReportDetail{
		ID:      id,
		Name:    strings.TrimSuffix(id, ext),
		Format:  format,
		Content: string(content),
	}, nil
}

// ListTestResults reads from the data/test-results.json file.
func (s *Store) ListTestResults() ([]TestResultSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results, err := s.readTestResults()
	if err != nil {
		return nil, err
	}

	var summaries []TestResultSummary
	for _, r := range results {
		summaries = append(summaries, TestResultSummary{
			ID:          r.ID,
			Name:        r.Name,
			Status:      r.Status,
			Timestamp:   r.Timestamp,
			Duration:    r.Duration,
			SuccessRate: r.SuccessRate,
		})
	}

	return summaries, nil
}

// GetTestResult finds a test by ID.
func (s *Store) GetTestResult(id string) (*TestResultDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results, err := s.readTestResults()
	if err != nil {
		return nil, err
	}

	for _, r := range results {
		if r.ID == id {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("test result not found: %s", id)
}

// SaveTestResult appends a new result.
func (s *Store) SaveTestResult(result TestResultDetail) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	results, _ := s.readTestResults()
	results = append(results, result)

	return s.writeTestResults(results)
}

// GetStats aggregates statistics from test results.
func (s *Store) GetStats() (*Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results, err := s.readTestResults()
	if err != nil {
		// Return empty stats if no data
		return &Stats{
			RecentTests: []TestResultSummary{},
			History:     []HistoryPoint{},
		}, nil
	}

	total := len(results)
	passed := 0
	failed := 0
	var totalRate float64

	for _, r := range results {
		if r.Status == "passed" {
			passed++
		} else {
			failed++
		}
		totalRate += r.SuccessRate
	}

	avgRate := 0.0
	if total > 0 {
		avgRate = totalRate / float64(total)
	}

	// Recent tests (last 10)
	var recent []TestResultSummary
	start := 0
	if len(results) > 10 {
		start = len(results) - 10
	}
	for i := len(results) - 1; i >= start; i-- {
		r := results[i]
		recent = append(recent, TestResultSummary{
			ID:          r.ID,
			Name:        r.Name,
			Status:      r.Status,
			Timestamp:   r.Timestamp,
			Duration:    r.Duration,
			SuccessRate: r.SuccessRate,
		})
	}

	// Build history (last 14 days)
	history := s.buildHistory(results, 14)

	return &Stats{
		TotalTests:     total,
		PassedTests:    passed,
		FailedTests:    failed,
		AvgDuration:    "42s",
		AvgSuccessRate: float64(int(avgRate*10)) / 10,
		RecentTests:    recent,
		History:        history,
	}, nil
}

// GetConfig reads the wizards-qa config file and redacts API keys.
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

		// Redact sensitive keys
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

func (s *Store) readTestResults() ([]TestResultDetail, error) {
	path := filepath.Join(s.dataDir, "test-results.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var file TestResultsFile
	if err := json.Unmarshal(data, &file); err != nil {
		// Try parsing as a plain array
		var results []TestResultDetail
		if err2 := json.Unmarshal(data, &results); err2 != nil {
			return nil, fmt.Errorf("parsing test results: %w", err)
		}
		return results, nil
	}

	return file.Results, nil
}

func (s *Store) writeTestResults(results []TestResultDetail) error {
	path := filepath.Join(s.dataDir, "test-results.json")

	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	file := TestResultsFile{
		Results: results,
		Updated: time.Now(),
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (s *Store) buildHistory(results []TestResultDetail, days int) []HistoryPoint {
	now := time.Now()
	dayMap := make(map[string]*HistoryPoint)

	for i := 0; i < days; i++ {
		d := now.AddDate(0, 0, -i)
		key := d.Format("Jan 02")
		dayMap[key] = &HistoryPoint{Date: key}
	}

	for _, r := range results {
		t, err := time.Parse(time.RFC3339, r.Timestamp)
		if err != nil {
			continue
		}
		key := t.Format("Jan 02")
		if pt, ok := dayMap[key]; ok {
			if r.Status == "passed" {
				pt.Passed++
			} else {
				pt.Failed++
			}
		}
	}

	var history []HistoryPoint
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		key := d.Format("Jan 02")
		if pt, ok := dayMap[key]; ok {
			history = append(history, *pt)
		}
	}

	// Sort to ensure chronological order
	sort.Slice(history, func(i, j int) bool {
		return i < j // already in order from loop
	})

	return history
}

// ListTemplates walks the flows directory and extracts {{VARIABLE}} patterns.
func (s *Store) ListTemplates() ([]TemplateInfo, error) {
	var templates []TemplateInfo

	err := filepath.Walk(s.flowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		name := strings.TrimSuffix(info.Name(), ext)
		category := "general"
		relDir := filepath.Dir(path)
		if relDir != s.flowsDir {
			category = filepath.Base(relDir)
		}
		relPath, _ := filepath.Rel(filepath.Dir(s.flowsDir), path)
		if relPath == "" {
			relPath = path
		}

		content, err := os.ReadFile(path)
		if err != nil {
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

// FlowsDir returns the flows directory path (for use by executor).
func (s *Store) FlowsDir() string {
	return s.flowsDir
}

// ListTestPlans reads from data/test-plans.json.
func (s *Store) ListTestPlans() ([]TestPlanSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plans, err := s.readTestPlans()
	if err != nil {
		return nil, err
	}

	var summaries []TestPlanSummary
	for _, p := range plans {
		summaries = append(summaries, TestPlanSummary{
			ID:        p.ID,
			Name:      p.Name,
			Status:    p.Status,
			FlowCount: len(p.FlowNames),
			CreatedAt: p.CreatedAt,
			LastRunID: p.LastRunID,
		})
	}

	return summaries, nil
}

// GetTestPlan finds a plan by ID.
func (s *Store) GetTestPlan(id string) (*TestPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plans, err := s.readTestPlans()
	if err != nil {
		return nil, err
	}

	for _, p := range plans {
		if p.ID == id {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("test plan not found: %s", id)
}

// SaveTestPlan creates or updates a plan.
func (s *Store) SaveTestPlan(plan TestPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plans, _ := s.readTestPlans()

	found := false
	for i, p := range plans {
		if p.ID == plan.ID {
			plans[i] = plan
			found = true
			break
		}
	}
	if !found {
		plans = append(plans, plan)
	}

	return s.writeTestPlans(plans)
}

// UpdateTestPlanStatus updates a plan's status and optional last run ID.
func (s *Store) UpdateTestPlanStatus(id, status, lastRunID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plans, err := s.readTestPlans()
	if err != nil {
		return err
	}

	for i, p := range plans {
		if p.ID == id {
			plans[i].Status = status
			if lastRunID != "" {
				plans[i].LastRunID = lastRunID
			}
			return s.writeTestPlans(plans)
		}
	}

	return fmt.Errorf("test plan not found: %s", id)
}

func (s *Store) readTestPlans() ([]TestPlan, error) {
	path := filepath.Join(s.dataDir, "test-plans.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var file TestPlansFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing test plans: %w", err)
	}

	return file.Plans, nil
}

func (s *Store) writeTestPlans(plans []TestPlan) error {
	path := filepath.Join(s.dataDir, "test-plans.json")

	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	file := TestPlansFile{
		Plans:   plans,
		Updated: time.Now(),
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
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
