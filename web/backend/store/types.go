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
	ProjectID   string  `json:"projectId,omitempty"`
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
	CreatedBy   string       `json:"createdBy,omitempty"`
	ProjectID   string       `json:"projectId,omitempty"`
	PlanID      string       `json:"planId,omitempty"`
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
	TotalAnalyses  int                 `json:"totalAnalyses"`
	TotalFlows     int                 `json:"totalFlows"`
	TotalPlans     int                 `json:"totalPlans"`
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

// TestResultsFile represents the JSON structure stored on disk (used for migration).
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
	FlowNames   []string          `json:"flowNames"`
	Variables   map[string]string `json:"variables"`
	Status      string            `json:"status"`
	CreatedAt   string            `json:"createdAt"`
	LastRunID   string            `json:"lastRunId,omitempty"`
	CreatedBy   string            `json:"createdBy,omitempty"`
	ProjectID   string            `json:"projectId,omitempty"`
	AnalysisID  string            `json:"analysisId,omitempty"`
	Mode        string            `json:"mode,omitempty"` // "agent" or "" (empty = legacy maestro/browser)
}

type TestPlanSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	FlowCount  int    `json:"flowCount"`
	CreatedAt  string `json:"createdAt"`
	LastRunID  string `json:"lastRunId,omitempty"`
	ProjectID  string `json:"projectId,omitempty"`
	AnalysisID string `json:"analysisId,omitempty"`
}

type TestPlansFile struct {
	Plans   []TestPlan `json:"plans"`
	Updated time.Time  `json:"updated"`
}

type AnalysisRecord struct {
	ID           string      `json:"id"`
	GameURL      string      `json:"gameUrl"`
	Status       string      `json:"status"`
	Step         string      `json:"step,omitempty"`
	UpdatedAt    string      `json:"updatedAt,omitempty"`
	Framework    string      `json:"framework"`
	GameName     string      `json:"gameName"`
	FlowCount    int         `json:"flowCount"`
	CreatedAt    string      `json:"createdAt"`
	Result       interface{} `json:"result,omitempty"`
	CreatedBy    string      `json:"createdBy,omitempty"`
	ProjectID    string      `json:"projectId,omitempty"`
	ErrorMessage  string      `json:"errorMessage,omitempty"`
	Modules       string      `json:"modules,omitempty"`
	PartialResult string      `json:"partialResult,omitempty"`
	AgentMode      bool        `json:"agentMode"`
	Profile        string      `json:"profile,omitempty"`
	LastTestRunID  string      `json:"lastTestRunId,omitempty"`
}

type AgentStepRecord struct {
	ID             int    `json:"id"`
	AnalysisID     string `json:"analysisId"`
	StepNumber     int    `json:"stepNumber"`
	ToolName       string `json:"toolName"`
	Input          string `json:"input"`
	Result         string `json:"result"`
	ScreenshotPath string `json:"screenshotPath,omitempty"`
	DurationMs     int    `json:"durationMs"`
	ThinkingMs     int    `json:"thinkingMs,omitempty"`
	Error          string `json:"error,omitempty"`
	Reasoning      string `json:"reasoning,omitempty"`
	CreatedAt      string `json:"createdAt"`
}

type AnalysesFile struct {
	Analyses []AnalysisRecord `json:"analyses"`
	Updated  time.Time        `json:"updated"`
}

// User represents a registered user.
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	DisplayName  string `json:"displayName"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
	CreatedAt    string `json:"createdAt"`
}

// UserSummary is a public-safe user representation (no password hash).
type UserSummary struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	CreatedAt   string `json:"createdAt"`
}

// Project represents a top-level organizational entity grouping analyses, test plans, and test results.
type Project struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	GameURL     string            `json:"gameUrl"`
	Description string            `json:"description"`
	Color       string            `json:"color"`
	Icon        string            `json:"icon"`
	Tags        []string          `json:"tags"`
	Settings    map[string]string `json:"settings"`
	CreatedBy   string            `json:"createdBy,omitempty"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

// ProjectSummary extends Project with aggregated counts for listing.
type ProjectSummary struct {
	Project
	AnalysisCount int `json:"analysisCount"`
	PlanCount     int `json:"planCount"`
	TestCount     int `json:"testCount"`
	MemberCount   int `json:"memberCount"`
}

// ProjectMember represents a user's membership in a project.
type ProjectMember struct {
	ID          string `json:"id"`
	ProjectID   string `json:"projectId"`
	UserID      string `json:"userId"`
	Role        string `json:"role"`
	CreatedAt   string `json:"createdAt"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}
