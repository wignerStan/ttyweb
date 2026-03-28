package ai

import "time"

// ConversationMessage represents a single message in an AI conversation.
type ConversationMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolUse    []ToolUseBlock   `json:"toolUse,omitempty"`
	ToolResult []ToolResultBlock `json:"toolResult,omitempty"`
	Timestamp  time.Time        `json:"timestamp"`
}

// ToolUseBlock represents a tool invocation by the AI assistant.
type ToolUseBlock struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input,omitempty"`
}

// ToolResultBlock represents the result of a tool invocation.
type ToolResultBlock struct {
	ToolUseID string `json:"toolUseId"`
	Content   string `json:"content"`
	IsError   bool   `json:"isError,omitempty"`
}
