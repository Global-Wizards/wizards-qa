package performance

import (
	"fmt"
	"time"
)

// Metrics represents performance metrics for a test run
type Metrics struct {
	TestName       string
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	FPS            []float64 // Frames per second samples
	LoadTime       time.Duration
	ResponseTimes  []time.Duration
	MemoryUsage    []int64 // Memory usage in bytes
	CPUUsage       []float64 // CPU usage percentage
	NetworkLatency []time.Duration
}

// NewMetrics creates a new performance metrics collector
func NewMetrics(testName string) *Metrics {
	return &Metrics{
		TestName:       testName,
		StartTime:      time.Now(),
		FPS:            []float64{},
		ResponseTimes:  []time.Duration{},
		MemoryUsage:    []int64{},
		CPUUsage:       []float64{},
		NetworkLatency: []time.Duration{},
	}
}

// RecordFPS records a frames-per-second measurement
func (m *Metrics) RecordFPS(fps float64) {
	m.FPS = append(m.FPS, fps)
}

// RecordResponseTime records a response time
func (m *Metrics) RecordResponseTime(duration time.Duration) {
	m.ResponseTimes = append(m.ResponseTimes, duration)
}

// RecordMemory records memory usage
func (m *Metrics) RecordMemory(bytes int64) {
	m.MemoryUsage = append(m.MemoryUsage, bytes)
}

// RecordCPU records CPU usage
func (m *Metrics) RecordCPU(percentage float64) {
	m.CPUUsage = append(m.CPUUsage, percentage)
}

// RecordNetworkLatency records network latency
func (m *Metrics) RecordNetworkLatency(duration time.Duration) {
	m.NetworkLatency = append(m.NetworkLatency, duration)
}

// Finalize marks the test as complete and calculates final metrics
func (m *Metrics) Finalize() {
	m.EndTime = time.Now()
	m.Duration = m.EndTime.Sub(m.StartTime)
}

// AverageFPS calculates average frames per second
func (m *Metrics) AverageFPS() float64 {
	if len(m.FPS) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, fps := range m.FPS {
		sum += fps
	}
	return sum / float64(len(m.FPS))
}

// MinFPS returns minimum FPS recorded
func (m *Metrics) MinFPS() float64 {
	if len(m.FPS) == 0 {
		return 0
	}
	
	min := m.FPS[0]
	for _, fps := range m.FPS {
		if fps < min {
			min = fps
		}
	}
	return min
}

// MaxFPS returns maximum FPS recorded
func (m *Metrics) MaxFPS() float64 {
	if len(m.FPS) == 0 {
		return 0
	}
	
	max := m.FPS[0]
	for _, fps := range m.FPS {
		if fps > max {
			max = fps
		}
	}
	return max
}

// AverageResponseTime calculates average response time
func (m *Metrics) AverageResponseTime() time.Duration {
	if len(m.ResponseTimes) == 0 {
		return 0
	}
	
	sum := time.Duration(0)
	for _, rt := range m.ResponseTimes {
		sum += rt
	}
	return sum / time.Duration(len(m.ResponseTimes))
}

// P95ResponseTime calculates 95th percentile response time
func (m *Metrics) P95ResponseTime() time.Duration {
	if len(m.ResponseTimes) == 0 {
		return 0
	}
	
	// Simple calculation - would need sorting for accurate percentile
	index := int(float64(len(m.ResponseTimes)) * 0.95)
	if index >= len(m.ResponseTimes) {
		index = len(m.ResponseTimes) - 1
	}
	return m.ResponseTimes[index]
}

// AverageMemory calculates average memory usage
func (m *Metrics) AverageMemory() int64 {
	if len(m.MemoryUsage) == 0 {
		return 0
	}
	
	sum := int64(0)
	for _, mem := range m.MemoryUsage {
		sum += mem
	}
	return sum / int64(len(m.MemoryUsage))
}

// PeakMemory returns peak memory usage
func (m *Metrics) PeakMemory() int64 {
	if len(m.MemoryUsage) == 0 {
		return 0
	}
	
	peak := m.MemoryUsage[0]
	for _, mem := range m.MemoryUsage {
		if mem > peak {
			peak = mem
		}
	}
	return peak
}

// AverageCPU calculates average CPU usage
func (m *Metrics) AverageCPU() float64 {
	if len(m.CPUUsage) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, cpu := range m.CPUUsage {
		sum += cpu
	}
	return sum / float64(len(m.CPUUsage))
}

// Summary returns a formatted summary of metrics
func (m *Metrics) Summary() string {
	return fmt.Sprintf(`Performance Metrics: %s
  Duration: %s
  Load Time: %s
  
  FPS:
    Average: %.1f
    Min: %.1f
    Max: %.1f
    Samples: %d
  
  Response Time:
    Average: %s
    P95: %s
    Samples: %d
  
  Memory:
    Average: %s
    Peak: %s
    Samples: %d
  
  CPU:
    Average: %.1f%%
    Samples: %d`,
		m.TestName,
		m.Duration.Round(time.Millisecond),
		m.LoadTime.Round(time.Millisecond),
		m.AverageFPS(),
		m.MinFPS(),
		m.MaxFPS(),
		len(m.FPS),
		m.AverageResponseTime().Round(time.Millisecond),
		m.P95ResponseTime().Round(time.Millisecond),
		len(m.ResponseTimes),
		formatBytes(m.AverageMemory()),
		formatBytes(m.PeakMemory()),
		len(m.MemoryUsage),
		m.AverageCPU(),
		len(m.CPUUsage),
	)
}

// formatBytes formats bytes into human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// PerformanceThresholds defines acceptable performance thresholds
type PerformanceThresholds struct {
	MinFPS          float64
	MaxLoadTime     time.Duration
	MaxResponseTime time.Duration
	MaxMemoryMB     int64
	MaxCPUPercent   float64
}

// DefaultThresholds returns sensible default performance thresholds
func DefaultThresholds() *PerformanceThresholds {
	return &PerformanceThresholds{
		MinFPS:          30.0,  // 30 FPS minimum
		MaxLoadTime:     5 * time.Second,
		MaxResponseTime: 100 * time.Millisecond,
		MaxMemoryMB:     512, // 512 MB
		MaxCPUPercent:   80.0,
	}
}

// Check verifies metrics against thresholds
func (m *Metrics) Check(thresholds *PerformanceThresholds) []string {
	issues := []string{}
	
	// Check FPS
	avgFPS := m.AverageFPS()
	if avgFPS > 0 && avgFPS < thresholds.MinFPS {
		issues = append(issues, fmt.Sprintf("Average FPS (%.1f) below threshold (%.1f)", avgFPS, thresholds.MinFPS))
	}
	
	// Check load time
	if m.LoadTime > thresholds.MaxLoadTime {
		issues = append(issues, fmt.Sprintf("Load time (%s) exceeds threshold (%s)", m.LoadTime, thresholds.MaxLoadTime))
	}
	
	// Check response time
	avgRT := m.AverageResponseTime()
	if avgRT > 0 && avgRT > thresholds.MaxResponseTime {
		issues = append(issues, fmt.Sprintf("Average response time (%s) exceeds threshold (%s)", avgRT, thresholds.MaxResponseTime))
	}
	
	// Check memory
	peakMem := m.PeakMemory() / (1024 * 1024) // Convert to MB
	if peakMem > thresholds.MaxMemoryMB {
		issues = append(issues, fmt.Sprintf("Peak memory (%d MB) exceeds threshold (%d MB)", peakMem, thresholds.MaxMemoryMB))
	}
	
	// Check CPU
	avgCPU := m.AverageCPU()
	if avgCPU > 0 && avgCPU > thresholds.MaxCPUPercent {
		issues = append(issues, fmt.Sprintf("Average CPU (%.1f%%) exceeds threshold (%.1f%%)", avgCPU, thresholds.MaxCPUPercent))
	}
	
	return issues
}
