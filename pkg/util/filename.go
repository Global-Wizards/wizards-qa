package util

import "strings"

// SanitizeFilename removes unsafe characters from filenames, replacing them with dashes.
// The result is lowercased for consistent file naming.
func SanitizeFilename(name string) string {
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	safe := strings.ToLower(name)
	for _, char := range unsafe {
		safe = strings.ReplaceAll(safe, char, "-")
	}
	return safe
}
