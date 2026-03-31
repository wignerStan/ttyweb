package ai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// jsonlScanner returns a buffered scanner for a JSONL file, or an error.
// The caller must close the returned file.
func jsonlScanner(filePath string) (*os.File, *bufio.Scanner, error) {
	file, err := os.Open(filePath) //nolint:gosec // reason: filePath comes from AI log directory, not user input
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file %q: %w", filePath, err)
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	return file, scanner, nil
}

// ParseClaudeConversation reads a Claude Code JSONL session file and extracts
// user and assistant messages, including tool use blocks.
func ParseClaudeConversation(filePath string) ([]ConversationMessage, error) {
	file, scanner, err := jsonlScanner(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var messages []ConversationMessage
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry struct {
			Type      string          `json:"type"`
			Message   json.RawMessage `json:"message"`
			Timestamp string          `json:"timestamp"`
			IsMeta    bool            `json:"isMeta"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		ts, _ := time.Parse(time.RFC3339, entry.Timestamp)

		var msgContent claudeMsgContent
		if err := json.Unmarshal(entry.Message, &msgContent); err != nil {
			continue
		}

		if msgs, ok := processClaudeEntry(entry.Type, entry.IsMeta, msgContent, ts); ok {
			messages = append(messages, msgs...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ParseClaudeConversation: scanning %q: %w", filePath, err)
	}
	return messages, nil
}

// claudeMsgContent is a Claude message content entry.
type claudeMsgContent struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// processClaudeEntry processes a single Claude JSONL entry, returning messages if applicable.
func processClaudeEntry(entryType string, isMeta bool, msgContent claudeMsgContent, ts time.Time) ([]ConversationMessage, bool) {
	switch entryType {
	case "user":
		if isMeta || msgContent.Role != "user" {
			return nil, false
		}
		return parseClaudeUserContent(msgContent.Content, ts), true
	case "assistant":
		if msgContent.Role != "assistant" {
			return nil, false
		}
		msg := parseClaudeAssistantContent(msgContent.Content, ts)
		if msg.Content != "" || len(msg.ToolUse) > 0 {
			return []ConversationMessage{msg}, true
		}
		return nil, false
	default:
		return nil, false
	}
}

// isSkippableContent returns true for empty strings or Claude Code internal commands.
func isSkippableContent(text string) bool {
	return text == "" ||
		strings.HasPrefix(text, "<command-") ||
		strings.HasPrefix(text, "<local-command")
}

// parseClaudeUserContent extracts messages from Claude user content.
// Content can be a plain string or an array of content blocks (including tool results).
func parseClaudeUserContent(content any, ts time.Time) []ConversationMessage {
	switch v := content.(type) {
	case string:
		text := strings.TrimSpace(v)
		if isSkippableContent(text) {
			return nil
		}
		return []ConversationMessage{{
			Role:      "user",
			Content:   text,
			Timestamp: ts,
		}}
	case []any:
		if hasClaudeToolResultInBlocks(v) {
			return parseClaudeToolResultBlocks(v, ts)
		}
		text := strings.TrimSpace(renderClaudeBlocksToText(v))
		if isSkippableContent(text) {
			return nil
		}
		return []ConversationMessage{{
			Role:      "user",
			Content:   text,
			Timestamp: ts,
		}}
	default:
		return nil
	}
}

// parseClaudeAssistantContent extracts messages from Claude assistant content.
func parseClaudeAssistantContent(content any, ts time.Time) ConversationMessage {
	msg := ConversationMessage{
		Role:      "assistant",
		Timestamp: ts,
	}

	switch v := content.(type) {
	case string:
		msg.Content = strings.TrimSpace(v)
	case []any:
		for _, block := range v {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := blockMap["type"].(string)

			switch blockType {
			case "text":
				if text, ok := blockMap["text"].(string); ok {
					msg.Content += text
				}
			case "tool_use":
				id, _ := blockMap["id"].(string)
				name, _ := blockMap["name"].(string)
				inputJSON, _ := json.Marshal(blockMap["input"])
				msg.ToolUse = append(msg.ToolUse, ToolUseBlock{
					ID:    id,
					Name:  name,
					Input: string(inputJSON),
				})
			}
		}
		msg.Content = strings.TrimSpace(msg.Content)
	}

	return msg
}

// hasClaudeToolResultInBlocks checks if content blocks contain tool_result blocks.
func hasClaudeToolResultInBlocks(blocks []any) bool {
	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := blockMap["type"].(string); t == "tool_result" {
			return true
		}
	}
	return false
}

// parseClaudeToolResultBlocks extracts tool result messages from content blocks.
func parseClaudeToolResultBlocks(blocks []any, ts time.Time) []ConversationMessage {
	messages := make([]ConversationMessage, 0, len(blocks))
	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := blockMap["type"].(string); t != "tool_result" {
			continue
		}

		toolUseID, _ := blockMap["tool_use_id"].(string)
		isError, _ := blockMap["is_error"].(bool)

		contentText := renderClaudeBlocksToText(blockMap["content"])
		messages = append(messages, ConversationMessage{
			Role:      "user",
			Content:   contentText,
			Timestamp: ts,
			ToolResult: []ToolResultBlock{{
				ToolUseID: toolUseID,
				Content:   contentText,
				IsError:   isError,
			}},
		})
	}
	return messages
}

// renderClaudeBlocksToText converts content blocks to plain text.
func renderClaudeBlocksToText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, block := range v {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := blockMap["type"].(string)
			if blockType == "text" {
				if text, ok := blockMap["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		if text, ok := v["text"].(string); ok {
			return text
		}
		if raw, ok := v["rawContent"].(string); ok {
			return raw
		}
		b, _ := json.Marshal(v)
		return string(b)
	default:
		return ""
	}
}

// ParseCodexConversation reads a Codex JSONL session file and extracts
// user and assistant messages.
func ParseCodexConversation(filePath string) ([]ConversationMessage, error) {
	file, scanner, err := jsonlScanner(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var messages []ConversationMessage
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry struct {
			Timestamp string          `json:"timestamp"`
			Type      string          `json:"type"`
			Payload   json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		ts, _ := time.Parse(time.RFC3339, entry.Timestamp)

		if entry.Type != "event_msg" {
			continue
		}

		var payload codexPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			continue
		}

		if msg, ok := codexPayloadToMessage(payload, ts); ok {
			messages = append(messages, msg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ParseCodexConversation: scanning %q: %w", filePath, err)
	}
	return messages, nil
}

// codexPayload is a Codex event message payload.
type codexPayload struct {
	MsgType string `json:"type"`
	Message string `json:"message"`
	Text    string `json:"text"`
}

// codexPayloadToMessage converts a Codex event payload to a conversation message.
func codexPayloadToMessage(payload codexPayload, ts time.Time) (ConversationMessage, bool) {
	switch payload.MsgType {
	case "user_message":
		if payload.Message == "" {
			return ConversationMessage{}, false
		}
		return ConversationMessage{Role: "user", Content: payload.Message, Timestamp: ts}, true
	case "agent_message":
		if payload.Message == "" {
			return ConversationMessage{}, false
		}
		return ConversationMessage{Role: "assistant", Content: payload.Message, Timestamp: ts}, true
	case "agent_reasoning":
		if payload.Text == "" {
			return ConversationMessage{}, false
		}
		return ConversationMessage{Role: "assistant", Content: payload.Text, Timestamp: ts}, true
	default:
		return ConversationMessage{}, false
	}
}
