package store

import "time"

type FlowInfo struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Path     string `json:"path"`
}

type FlowDetail struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Path     string `json:"path"`
	Content  string `json:"content"`
}

type ReportInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Format    string `json:"format"`
	Timestamp string `json:"timestamp"`
	Size      string `json:"size"`
}

type ReportDetail struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Format  string `json:"format"`
	Content string `json:"content"`
}

type FlowResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
}

type TestResultSummary struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	Timestamp   string  `json:"timestamp"`
	Duration    string  `json:"duration"`
	SuccessRate float64 `json:"successRate"`
}

type TestResultDetail struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	Timestamp   string       `json:"timestamp"`
	Duration    string       `json:"duration"`
	SuccessRate float64      `json:"successRate"`
	Flows       []FlowResult `json:"flows,omitempty"`
	ErrorOutput string       `json:"errorOutput,omitempty"`
}

type HistoryPoint struct {
	Date   string `json:"date"`
	Passed int    `json:"passed"`
	Failed int    `json:"failed"`
}

type Stats struct {
	TotalTests     int                 `json:"totalTests"`
	PassedTests    int                 `json:"passedTests"`
	FailedTests    int                 `json:"failedTests"`
	AvgDuration    string              `json:"avgDuration"`
	AvgSuccessRate float64             `json:"avgSuccessRate"`
	RecentTests    []TestResultSummary `json:"recentTests"`
	History        []HistoryPoint      `json:"history"`
}

type ConfigData struct {
	GameURL     string            `json:"gameUrl,omitempty"`
	AIProvider  string            `json:"aiProvider,omitempty"`
	AIModel     string            `json:"aiModel,omitempty"`
	OutputDir   string            `json:"outputDir,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
	ExtraConfig map[string]string `json:"extraConfig,omitempty"`
}

// TestResultsFile represents the JSON structure stored on disk.
type TestResultsFile struct {
	Results []TestResultDetail `json:"results"`
	Updated time.Time          `json:"updated"`
}

type TemplateInfo struct {
	Name      string   `json:"name"`
	Category  string   `json:"category"`
	Path      string   `json:"path"`
	Variables []string `json:"variables"`
}

type TestPlan struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	GameURL     string            `json:"gameUrl"`
	FlowNames  []string          `json:"flowNames"`
	Variables   map[string]string `json:"variables"`
	Status      string            `json:"status"`
	CreatedAt   string            `json:"createdAt"`
	LastRunID   string            `json:"lastRunId,omitempty"`
}

type TestPlanSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	FlowCount int    `json:"flowCount"`
	CreatedAt string `json:"createdAt"`
	LastRunID string `json:"lastRunId,omitempty"`
}

type TestPlansFile struct {
	Plans   []TestPlan `json:"plans"`
	Updated time.Time  `json:"updated"`
}
