package ai

import "strings"

// BuildAnalysisPrompt constructs the prompt for game analysis.
func BuildAnalysisPrompt(prompt string, context map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString("You are an expert QA engineer specializing in game testing.\n")
	sb.WriteString("Analyze the provided game and respond with structured JSON.\n\n")

	if spec, ok := context["spec"].(string); ok {
		sb.WriteString("Game Specification:\n")
		sb.WriteString(spec)
		sb.WriteString("\n\n")
	}

	if url, ok := context["url"].(string); ok {
		sb.WriteString("Game URL: ")
		sb.WriteString(url)
		sb.WriteString("\n\n")
	}

	sb.WriteString(prompt)

	return sb.String()
}

// BuildGenerationPrompt constructs the prompt for flow generation.
func BuildGenerationPrompt(prompt string, context map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString("You are an expert at creating Maestro test flows for game testing.\n")
	sb.WriteString("Generate valid Maestro YAML flows based on the provided requirements.\n\n")

	if analysis, ok := context["analysis"].(string); ok {
		sb.WriteString("Game Analysis:\n")
		sb.WriteString(analysis)
		sb.WriteString("\n\n")
	}

	sb.WriteString(prompt)

	return sb.String()
}
