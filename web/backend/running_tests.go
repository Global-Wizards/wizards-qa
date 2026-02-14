package main

import (
	"sync"

	"github.com/Global-Wizards/wizards-qa/web/backend/store"
)

// RunningTestTracker manages the shared runningTests map with proper synchronization.
type RunningTestTracker struct {
	mu    sync.Mutex
	tests map[string]*runningTest
}

// NewRunningTestTracker creates a new tracker.
func NewRunningTestTracker() *RunningTestTracker {
	return &RunningTestTracker{
		tests: make(map[string]*runningTest),
	}
}

// Register adds a running test to the tracker.
func (t *RunningTestTracker) Register(testID string, rt *runningTest) {
	t.mu.Lock()
	t.tests[testID] = rt
	t.mu.Unlock()
}

// AppendLog adds a log line to a running test's log buffer, maintaining the max size.
func (t *RunningTestTracker) AppendLog(testID, line string) {
	t.mu.Lock()
	if rt, ok := t.tests[testID]; ok {
		if len(rt.Logs) >= maxRunningTestLogs {
			rt.Logs = rt.Logs[1:]
		}
		rt.Logs = append(rt.Logs, line)
	}
	t.mu.Unlock()
}

// AppendFlow adds a flow result to a running test.
func (t *RunningTestTracker) AppendFlow(testID string, fr store.FlowResult) {
	t.mu.Lock()
	if rt, ok := t.tests[testID]; ok {
		rt.Flows = append(rt.Flows, fr)
	}
	t.mu.Unlock()
}

// Remove removes a test from the tracker.
func (t *RunningTestTracker) Remove(testID string) {
	t.mu.Lock()
	delete(t.tests, testID)
	t.mu.Unlock()
}

// Get returns a snapshot of the running test, or nil if not found.
func (t *RunningTestTracker) Get(testID string) *runningTest {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tests[testID]
}

// GetAll returns a snapshot of all running tests.
func (t *RunningTestTracker) GetAll() map[string]*runningTest {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make(map[string]*runningTest, len(t.tests))
	for k, v := range t.tests {
		result[k] = v
	}
	return result
}
