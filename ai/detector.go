package ai

import "strings"

// AssistantType represents the type of AI assistant detected.
type AssistantType string

const (
	AssistantTypeUnknown    AssistantType = ""
	AssistantTypeClaudeCode AssistantType = "claude_code"
	AssistantTypeCodex      AssistantType = "codex"
)

// String returns the string representation of the assistant type.
func (t AssistantType) String() string {
	return string(t)
}

// DisplayName returns a human-readable name for the assistant type.
func (t AssistantType) DisplayName() string {
	switch t {
	case AssistantTypeClaudeCode:
		return "Claude Code"
	case AssistantTypeCodex:
		return "OpenAI Codex"
	case AssistantTypeUnknown:
		return ""
	default:
		return string(t)
	}
}

// detectionRule defines how to detect a specific AI assistant from a command string.
type detectionRule struct {
	assistantType AssistantType
	patterns      []string
}

var detectionRules = []detectionRule{
	{
		assistantType: AssistantTypeClaudeCode,
		patterns: []string{
			"@anthropic-ai/claude-code",
			"claude-code/cli.js",
			"claude-code/bin/",
			"/claude",
		},
	},
	{
		assistantType: AssistantTypeCodex,
		patterns: []string{
			"@openai/codex",
			"codex/bin/codex.js",
			"codex.js",
			"/codex",
		},
	},
}

// DetectAssistant checks if a command string indicates running Claude Code or Codex.
// Returns the detected assistant type and true if detected, or empty string and false otherwise.
func DetectAssistant(command string) (AssistantType, bool) {
	if command == "" {
		return AssistantTypeUnknown, false
	}

	normalized := strings.ToLower(command)
	normalized = strings.ReplaceAll(normalized, "\\", "/")

	for _, rule := range detectionRules {
		for _, pattern := range rule.patterns {
			if strings.Contains(normalized, pattern) {
				return rule.assistantType, true
			}
		}
	}

	return AssistantTypeUnknown, false
}
