package ai

import "strings"

// AssistantType represents the type of AI assistant detected.
type AssistantType string

const (
	// AssistantTypeUnknown represents an undetermined assistant type.
	AssistantTypeUnknown AssistantType = ""
	// AssistantTypeClaudeCode represents a Claude Code assistant.
	AssistantTypeClaudeCode AssistantType = "claude_code"
	// AssistantTypeCodex represents a Codex assistant.
	AssistantTypeCodex AssistantType = "codex"
	// AssistantTypeQwenCode represents a Qwen Code assistant.
	AssistantTypeQwenCode AssistantType = "qwen_code"
	// AssistantTypeGemini represents a Gemini assistant.
	AssistantTypeGemini AssistantType = "gemini"
	// AssistantTypeCursor represents a Cursor assistant.
	AssistantTypeCursor AssistantType = "cursor"
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
	case AssistantTypeQwenCode:
		return "Qwen Code"
	case AssistantTypeGemini:
		return "Gemini"
	case AssistantTypeCursor:
		return "Cursor"
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
	{
		assistantType: AssistantTypeQwenCode,
		patterns: []string{
			"@alicloud/qwen-code",
			"qwen-code",
			"qwen_code",
			"/qwen",
		},
	},
	{
		assistantType: AssistantTypeGemini,
		patterns: []string{
			"@google/gemini-cli",
			"gemini-cli",
			"/gemini",
		},
	},
	{
		assistantType: AssistantTypeCursor,
		patterns: []string{
			"cursor-agent",
			"/cursor",
			".cursor/",
		},
	},
}

// DetectAssistant checks if a command string indicates running an AI assistant.
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
