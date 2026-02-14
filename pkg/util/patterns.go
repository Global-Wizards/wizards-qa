package util

import "regexp"

// VarPattern matches {{VAR}} template placeholders.
var VarPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

// SafeNameRegex matches names that are safe for use as filenames/identifiers.
// Allows alphanumeric, underscores, hyphens, spaces, and periods.
var SafeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\s.]+$`)
