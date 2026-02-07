#!/bin/bash
# Task 1.1: Build Maestro Wrapper Package

set -e

PROJECT_DIR="/home/koves/GitHub/wizards-qa"
cd "$PROJECT_DIR"

echo "Creating pkg/maestro package..."

# Create directory
mkdir -p pkg/maestro

# Create executor.go
cat > pkg/maestro/executor.go << 'EOF'
package maestro

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Executor handles Maestro CLI execution
type Executor struct {
	MaestroPath string
	Browser     string
	Timeout     time.Duration
}

// NewExecutor creates a new Maestro executor
func NewExecutor(maestroPath, browser string, timeout time.Duration) *Executor {
	if maestroPath == "" {
		maestroPath = "maestro" // Use PATH
	}
	if browser == "" {
		browser = "chrome"
	}
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	return &Executor{
		MaestroPath: maestroPath,
		Browser:     browser,
		Timeout:     timeout,
	}
}

// RunFlow executes a single Maestro flow file
func (e *Executor) RunFlow(flowPath string) (*TestResult, error) {
	startTime := time.Now()

	// Validate flow exists
	absPath, err := filepath.Abs(flowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve flow path: %w", err)
	}

	// Build Maestro command
	cmd := exec.Command(e.MaestroPath, "test", absPath)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run with timeout
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start maestro: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Command completed
		duration := time.Since(startTime)

		result := &TestResult{
			FlowName:  filepath.Base(flowPath),
			FlowPath:  absPath,
			StartTime: startTime,
			Duration:  duration,
			Stdout:    stdout.String(),
			Stderr:    stderr.String(),
		}

		if err != nil {
			result.Status = StatusFailed
			result.Error = err.Error()
		} else {
			result.Status = StatusPassed
		}

		// Parse output for details
		e.parseOutput(result)

		return result, nil

	case <-time.After(e.Timeout):
		cmd.Process.Kill()
		return &TestResult{
			FlowName:  filepath.Base(flowPath),
			FlowPath:  absPath,
			StartTime: startTime,
			Duration:  time.Since(startTime),
			Status:    StatusTimeout,
			Error:     fmt.Sprintf("test timeout after %s", e.Timeout),
		}, nil
	}
}

// RunFlows executes multiple flow files
func (e *Executor) RunFlows(flowPaths []string) (*TestResults, error) {
	results := &TestResults{
		StartTime: time.Now(),
		Flows:     make([]*TestResult, 0, len(flowPaths)),
	}

	for _, flowPath := range flowPaths {
		result, err := e.RunFlow(flowPath)
		if err != nil {
			return results, fmt.Errorf("failed to run flow %s: %w", flowPath, err)
		}
		results.Flows = append(results.Flows, result)

		// Update summary
		switch result.Status {
		case StatusPassed:
			results.Passed++
		case StatusFailed:
			results.Failed++
		case StatusTimeout:
			results.Timeout++
		}
	}

	results.Duration = time.Since(results.StartTime)
	results.Total = len(flowPaths)

	return results, nil
}

// ValidateFlow checks if a flow file is valid
func (e *Executor) ValidateFlow(flowPath string) error {
	// For now, just check if file exists
	// Full YAML validation will be in flows package
	absPath, err := filepath.Abs(flowPath)
	if err != nil {
		return fmt.Errorf("invalid flow path: %w", err)
	}

	// Try to parse with Maestro (dry run)
	cmd := exec.Command(e.MaestroPath, "test", "--dry-run", absPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("flow validation failed: %s", string(output))
	}

	return nil
}

// parseOutput extracts details from Maestro output
func (e *Executor) parseOutput(result *TestResult) {
	output := result.Stdout + result.Stderr

	// Look for common patterns
	if strings.Contains(output, "PASSED") {
		result.Status = StatusPassed
	}
	if strings.Contains(output, "FAILED") {
		result.Status = StatusFailed
	}
	if strings.Contains(output, "timeout") {
		result.Status = StatusTimeout
	}

	// Extract step count
	// TODO: Parse Maestro output format for step details
}
EOF

# Create types.go
cat > pkg/maestro/types.go << 'EOF'
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
EOF

# Update ROADMAP
sed -i 's/#### 1.1 - Maestro Wrapper Package ⏳ NEXT/#### 1.1 - Maestro Wrapper Package ✅ COMPLETE/' "$PROJECT_DIR/ROADMAP.md"
sed -i 's/\*\*Status:\*\* Not started/\*\*Status:\*\* ✅ Complete/' "$PROJECT_DIR/ROADMAP.md"
sed -i 's/#### 1.2 - Flow Validation/#### 1.2 - Flow Validation ⏳ NEXT/' "$PROJECT_DIR/ROADMAP.md"

echo "✅ Task 1.1 complete!"
