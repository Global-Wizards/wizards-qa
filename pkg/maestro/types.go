package maestro

import "time"

// Status represents test execution status
type Status string

const (
	StatusPassed  Status = "passed"
	StatusFailed  Status = "failed"
	StatusTimeout Status = "timeout"
	StatusError   Status = "error"
)

// TestResult represents the result of running a single flow
type TestResult struct {
	FlowName  string        `json:"flowName"`
	FlowPath  string        `json:"flowPath"`
	Status    Status        `json:"status"`
	StartTime time.Time     `json:"startTime"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
	Stdout    string        `json:"stdout,omitempty"`
	Stderr    string        `json:"stderr,omitempty"`
	Steps     int           `json:"steps,omitempty"`
}

// TestResults represents results from multiple flows
type TestResults struct {
	Total     int           `json:"total"`
	Passed    int           `json:"passed"`
	Failed    int           `json:"failed"`
	Timeout   int           `json:"timeout"`
	StartTime time.Time     `json:"startTime"`
	Duration  time.Duration `json:"duration"`
	Flows     []*TestResult `json:"flows"`
}

// SuccessRate returns the percentage of passed tests
func (r *TestResults) SuccessRate() float64 {
	if r.Total == 0 {
		return 0
	}
	return float64(r.Passed) / float64(r.Total) * 100
}
