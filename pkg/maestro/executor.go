package maestro

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Global-Wizards/wizards-qa/pkg/parallel"
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
			if stderrStr := strings.TrimSpace(stderr.String()); stderrStr != "" {
				result.Error = fmt.Sprintf("%s\n%s", err.Error(), stderrStr)
			} else {
				result.Error = err.Error()
			}
		} else {
			result.Status = StatusPassed
		}

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
	return e.RunFlowsWithOptions(flowPaths, nil)
}

// RunFlowsWithOptions executes flows with custom options
func (e *Executor) RunFlowsWithOptions(flowPaths []string, opts *ExecutionOptions) (*TestResults, error) {
	if opts == nil {
		opts = DefaultExecutionOptions()
	}

	results := &TestResults{
		StartTime: time.Now(),
		Flows:     make([]*TestResult, len(flowPaths)),
	}

	if opts.Parallel {
		// Parallel execution
		tasks := make([]parallel.Task, len(flowPaths))
		for i, path := range flowPaths {
			p := path
			idx := i
			tasks[i] = func() error {
				result, err := e.RunFlow(p)
				if err != nil {
					return err
				}
				results.Flows[idx] = result
				return nil
			}
		}
		
		errors := parallel.Execute(context.Background(), tasks, opts.MaxConcurrency)
		for _, err := range errors {
			if err != nil {
				return results, fmt.Errorf("parallel execution failed: %w", err)
			}
		}
	} else {
		// Sequential execution
		for i, flowPath := range flowPaths {
			result, err := e.RunFlow(flowPath)
			if err != nil {
				return results, fmt.Errorf("failed to run flow %s: %w", flowPath, err)
			}
			results.Flows[i] = result
		}
	}

	// Update summary
	for _, result := range results.Flows {
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

// ExecutionOptions configures flow execution
type ExecutionOptions struct {
	Parallel       bool
	MaxConcurrency int
	FailFast       bool
	Retry          bool
	RetryAttempts  int
}

// DefaultExecutionOptions returns sensible defaults
func DefaultExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		Parallel:       false,
		MaxConcurrency: 4,
		FailFast:       false,
		Retry:          false,
		RetryAttempts:  1,
	}
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

