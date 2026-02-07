package maestro

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CaptureManager manages screenshots and video capture for test runs
type CaptureManager struct {
	BaseDir string
	RunID   string
}

// NewCaptureManager creates a new capture manager
func NewCaptureManager(baseDir string) *CaptureManager {
	runID := fmt.Sprintf("run-%s", time.Now().Format("20060102-150405"))
	
	return &CaptureManager{
		BaseDir: baseDir,
		RunID:   runID,
	}
}

// PrepareDirectories creates necessary directories for capturing assets
func (cm *CaptureManager) PrepareDirectories() error {
	// Create base directory
	if err := os.MkdirAll(cm.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create run-specific directory
	runDir := cm.GetRunDir()
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return fmt.Errorf("failed to create run directory: %w", err)
	}

	// Create subdirectories
	dirs := []string{"screenshots", "videos", "logs"}
	for _, dir := range dirs {
		path := filepath.Join(runDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	return nil
}

// GetRunDir returns the directory for the current test run
func (cm *CaptureManager) GetRunDir() string {
	return filepath.Join(cm.BaseDir, cm.RunID)
}

// GetScreenshotDir returns the screenshots directory for the current run
func (cm *CaptureManager) GetScreenshotDir() string {
	return filepath.Join(cm.GetRunDir(), "screenshots")
}

// GetVideoDir returns the videos directory for the current run
func (cm *CaptureManager) GetVideoDir() string {
	return filepath.Join(cm.GetRunDir(), "videos")
}

// GetLogDir returns the logs directory for the current run
func (cm *CaptureManager) GetLogDir() string {
	return filepath.Join(cm.GetRunDir(), "logs")
}

// GetScreenshotPath returns the path for a specific screenshot
func (cm *CaptureManager) GetScreenshotPath(flowName, screenshotName string) string {
	safeFlowName := sanitizeFilename(flowName)
	return filepath.Join(cm.GetScreenshotDir(), fmt.Sprintf("%s-%s", safeFlowName, screenshotName))
}

// GetVideoPath returns the path for a specific video
func (cm *CaptureManager) GetVideoPath(flowName string) string {
	safeFlowName := sanitizeFilename(flowName)
	return filepath.Join(cm.GetVideoDir(), fmt.Sprintf("%s.mp4", safeFlowName))
}

// GetLogPath returns the path for a specific log file
func (cm *CaptureManager) GetLogPath(flowName string) string {
	safeFlowName := sanitizeFilename(flowName)
	return filepath.Join(cm.GetLogDir(), fmt.Sprintf("%s.log", safeFlowName))
}

// sanitizeFilename removes unsafe characters from filenames
func sanitizeFilename(name string) string {
	// Remove file extension if present
	if ext := filepath.Ext(name); ext != "" {
		name = name[:len(name)-len(ext)]
	}

	// Replace unsafe characters
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	safe := name
	for _, char := range unsafe {
		safe = filepath.Clean(safe)
		safe = filepath.ToSlash(safe)
		// Simple replacement with dash
		for i := 0; i < len(safe); i++ {
			if string(safe[i]) == char {
				safe = safe[:i] + "-" + safe[i+1:]
			}
		}
	}

	return safe
}

// CleanupOldRuns removes test runs older than the specified duration
func CleanupOldRuns(baseDir string, maxAge time.Duration) error {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, nothing to clean
		}
		return fmt.Errorf("failed to read directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		path := filepath.Join(baseDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.RemoveAll(path); err != nil {
				fmt.Printf("Warning: failed to remove old run %s: %v\n", entry.Name(), err)
			}
		}
	}

	return nil
}
