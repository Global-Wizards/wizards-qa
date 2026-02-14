package flows

import (
	"fmt"
	"regexp"
	"strings"
)

// OpenLinkObjRegex matches multi-line openLink blocks: openLink:\n  url: "..."
var OpenLinkObjRegex = regexp.MustCompile(`(?m)^(\s*- openLink):\s*\n\s+url:\s*"?([^"\n]+)"?\s*$`)

// ExtendedWaitTimeoutOnlyRegex matches extendedWaitUntil blocks with only a timeout (invalid).
var ExtendedWaitTimeoutOnlyRegex = regexp.MustCompile(`(?m)^[ \t]*- extendedWaitUntil:\s*\n[ \t]+timeout:\s*\d+\s*$`)

// SelectorVisibleRegex matches any command followed by visible: on the next indented line.
var SelectorVisibleRegex = regexp.MustCompile(`(?m)^(\s*- \w+):.*\n\s+visible:\s*"?([^"\n]+)"?\s*$`)

// SelectorNotVisibleRegex — same but for notVisible:
var SelectorNotVisibleRegex = regexp.MustCompile(`(?m)^(\s*- \w+):.*\n\s+notVisible:\s*"?([^"\n]+)"?\s*$`)

// BareVisibleRegex matches a bare "visible:" line NOT preceded by extendedWaitUntil or tapOn,
// wrapping it in an extendedWaitUntil block.
var BareVisibleRegex = regexp.MustCompile(`(?m)^(\s*)- visible:\s*"?([^"\n]+)"?\s*$`)

// MaestroCommandAliases maps invalid/old command names to correct Maestro names.
var MaestroCommandAliases = map[string]string{
	"waitFor":     "extendedWaitUntil",
	"screenshot":  "takeScreenshot",
	"openBrowser": "openLink",
}

// StripInvalidVisibleLines removes blank lines from the commands section and
// strips visible:/notVisible: lines from non-extendedWaitUntil command blocks.
func StripInvalidVisibleLines(content string) string {
	parts := strings.SplitN(content, "---\n", 2)
	var meta, cmds string
	if len(parts) == 2 {
		meta = parts[0] + "---\n"
		cmds = parts[1]
	} else {
		cmds = content
	}

	lines := strings.Split(cmds, "\n")
	var out []string
	inExtendedWait := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(line, "- extendedWaitUntil") {
			inExtendedWait = true
			out = append(out, line)
			continue
		}
		if strings.Contains(line, "- ") && !strings.HasPrefix(trimmed, "visible:") && !strings.HasPrefix(trimmed, "notVisible:") && !strings.HasPrefix(trimmed, "timeout:") && !strings.HasPrefix(trimmed, "point:") {
			inExtendedWait = false
		}
		if !inExtendedWait && (strings.HasPrefix(trimmed, "visible:") || strings.HasPrefix(trimmed, "notVisible:")) {
			continue
		}
		out = append(out, line)
	}

	return meta + strings.Join(out, "\n")
}

// NormalizeFlowYAML applies regex-based safety-net fixes to generated YAML.
// This catches issues that the structured serialization may miss.
func NormalizeFlowYAML(content string) string {
	result := StripInvalidVisibleLines(content)
	result = OpenLinkObjRegex.ReplaceAllString(result, `$1: "$2"`)
	skipExtended := func(match string) string {
		if strings.Contains(match, "extendedWaitUntil") {
			return match
		}
		return SelectorVisibleRegex.ReplaceAllString(match, `$1: "$2"`)
	}
	result = SelectorVisibleRegex.ReplaceAllStringFunc(result, skipExtended)
	skipExtendedNV := func(match string) string {
		if strings.Contains(match, "extendedWaitUntil") {
			return match
		}
		return SelectorNotVisibleRegex.ReplaceAllString(match, `$1: "$2"`)
	}
	result = SelectorNotVisibleRegex.ReplaceAllStringFunc(result, skipExtendedNV)
	for old, correct := range MaestroCommandAliases {
		result = strings.ReplaceAll(result, "- "+old+":", "- "+correct+":")
	}
	result = ExtendedWaitTimeoutOnlyRegex.ReplaceAllString(result, "")
	// Wrap bare "visible:" lines in extendedWaitUntil blocks
	result = BareVisibleRegex.ReplaceAllString(result, "${1}- extendedWaitUntil:\n${1}    visible: \"$2\"")
	return result
}

// FixCommandData fixes AI mistakes at the data level before yaml.Marshal.
// It translates command aliases, flattens invalid nested structures, strips
// newlines from string values, and removes invalid extendedWaitUntil blocks.
func FixCommandData(cmd map[string]interface{}, aliases map[string]string) map[string]interface{} {
	fixed := make(map[string]interface{})
	for key, value := range cmd {
		if key == "comment" {
			continue
		}
		if corrected, ok := aliases[key]; ok {
			key = corrected
		}
		switch v := value.(type) {
		case string:
			fixed[key] = strings.ReplaceAll(v, "\n", " ")
		case map[string]interface{}:
			if key == "openLink" {
				if urlVal, ok := v["url"]; ok {
					fixed[key] = strings.ReplaceAll(fmt.Sprintf("%v", urlVal), "\n", " ")
					continue
				}
			}
			if key != "extendedWaitUntil" {
				_, hasVis := v["visible"]
				_, hasNV := v["notVisible"]
				if hasVis || hasNV {
					if hasVis && len(v) == 1 {
						fixed[key] = strings.ReplaceAll(fmt.Sprintf("%v", v["visible"]), "\n", " ")
						continue
					}
					if hasNV && len(v) == 1 {
						fixed[key] = strings.ReplaceAll(fmt.Sprintf("%v", v["notVisible"]), "\n", " ")
						continue
					}
					delete(v, "visible")
					delete(v, "notVisible")
				}
			}
			if key == "extendedWaitUntil" {
				_, hasVisible := v["visible"]
				_, hasNotVisible := v["notVisible"]
				if !hasVisible && !hasNotVisible {
					continue
				}
			}
			cleanedSub := make(map[string]interface{})
			for sk, sv := range v {
				switch subV := sv.(type) {
				case string:
					cleanedSub[sk] = strings.ReplaceAll(subV, "\n", " ")
				case []interface{}:
					cleanedSub[sk] = FixCommandList(subV, aliases)
				default:
					cleanedSub[sk] = sv
				}
			}
			fixed[key] = cleanedSub
		case []interface{}:
			fixed[key] = FixCommandList(v, aliases)
		default:
			fixed[key] = value
		}
	}
	if len(fixed) == 0 {
		return nil
	}
	return fixed
}

// SplitVisibleFromCommand extracts top-level visible/notVisible from a non-extendedWaitUntil
// command and returns the original command (cleaned) plus a separate extendedWaitUntil command.
func SplitVisibleFromCommand(cmd map[string]interface{}) []map[string]interface{} {
	if _, isExtWait := cmd["extendedWaitUntil"]; isExtWait {
		return []map[string]interface{}{cmd}
	}
	vis, hasVis := cmd["visible"]
	nv, hasNV := cmd["notVisible"]
	if !hasVis && !hasNV {
		return []map[string]interface{}{cmd}
	}

	cleaned := make(map[string]interface{})
	for k, v := range cmd {
		if k != "visible" && k != "notVisible" {
			cleaned[k] = v
		}
	}

	waitInner := make(map[string]interface{})
	if hasVis {
		waitInner["visible"] = vis
	}
	if hasNV {
		waitInner["notVisible"] = nv
	}
	waitCmd := map[string]interface{}{"extendedWaitUntil": waitInner}

	return []map[string]interface{}{cleaned, waitCmd}
}

// FixCommandList recursively fixes a list of command maps (e.g. repeat.commands).
func FixCommandList(items []interface{}, aliases map[string]string) []interface{} {
	result := make([]interface{}, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			if fixed := FixCommandData(m, aliases); fixed != nil {
				result = append(result, fixed)
			}
		} else {
			result = append(result, item)
		}
	}
	return result
}
