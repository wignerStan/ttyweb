package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderClaudeBlocksToText_StringInput tests the string case.
func TestRenderClaudeBlocksToText_StringInput(t *testing.T) {
	got := renderClaudeBlocksToText("hello world")
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

// TestRenderClaudeBlocksToText_NilInput tests the nil case.
func TestRenderClaudeBlocksToText_NilInput(t *testing.T) {
	got := renderClaudeBlocksToText(nil)
	if got != "" {
		t.Errorf("expected empty string for nil, got %q", got)
	}
}

// TestRenderClaudeBlocksToText_IntInput tests the int case.
func TestRenderClaudeBlocksToText_IntInput(t *testing.T) {
	got := renderClaudeBlocksToText(42)
	if got != "" {
		t.Errorf("expected empty string for int, got %q", got)
	}
}

// TestRenderClaudeBlocksToText_SliceWithNonBlocks tests slice with non-map items.
func TestRenderClaudeBlocksToText_SliceWithNonBlocks(t *testing.T) {
	got := renderClaudeBlocksToText([]any{"string", 42, nil})
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// TestRenderClaudeBlocksToText_SliceWithTextBlock tests text block extraction.
func TestRenderClaudeBlocksToText_SliceWithTextBlock(t *testing.T) {
	got := renderClaudeBlocksToText([]any{
		map[string]any{"type": "text", "text": "block1"},
		map[string]any{"type": "text", "text": "block2"},
	})
	if got != "block1\nblock2" {
		t.Errorf("expected 'block1\\nblock2', got %q", got)
	}
}

// TestRenderClaudeBlocksToText_SliceWithNonTextBlock tests non-text blocks are skipped.
func TestRenderClaudeBlocksToText_SliceWithNonTextBlock(t *testing.T) {
	got := renderClaudeBlocksToText([]any{
		map[string]any{"type": "tool_use", "text": "block1"},
		map[string]any{"type": "text", "text": "block2"},
	})
	if got != "block2" {
		t.Errorf("expected 'block2', got %q", got)
	}
}

// TestRenderClaudeBlocksToText_MapWithRawContent tests rawContent fallback.
func TestRenderClaudeBlocksToText_MapWithRawContent(t *testing.T) {
	got := renderClaudeBlocksToText(map[string]any{"rawContent": "raw data"})
	if got != "raw data" {
		t.Errorf("expected 'raw data', got %q", got)
	}
}

// TestRenderClaudeBlocksToText_MapFallback tests JSON fallback.
func TestRenderClaudeBlocksToText_MapFallback(t *testing.T) {
	m := map[string]any{"key": "value"}
	got := renderClaudeBlocksToText(m)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	if got != string(data) {
		t.Errorf("expected JSON fallback, got %q", got)
	}
}

// TestParseCodexConversation_EmptyMessage tests empty message filtering.
func TestParseCodexConversation_EmptyMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	data := `{"timestamp":"2026-01-01T00:00:00Z","type":"event_msg","payload":{"type":"user_message","message":""}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	msgs, err := ParseCodexConversation(path)
	if err != nil {
		t.Fatalf("ParseCodexConversation: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for empty message, got %d", len(msgs))
	}
}

// TestParseCodexConversation_AgentReasoning tests agent_reasoning type.
func TestParseCodexConversation_AgentReasoning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	data := `{"timestamp":"2026-01-01T00:00:00Z","type":"event_msg","payload":{"type":"agent_reasoning","text":"thinking..."}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	msgs, err := ParseCodexConversation(path)
	if err != nil {
		t.Fatalf("ParseCodexConversation: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", msgs[0].Role)
	}
	if msgs[0].Content != "thinking..." {
		t.Errorf("expected content 'thinking...', got %q", msgs[0].Content)
	}
}

// TestParseCodexConversation_AgentMessageEmpty tests empty agent message.
func TestParseCodexConversation_AgentMessageEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	data := `{"timestamp":"2026-01-01T00:00:00Z","type":"event_msg","payload":{"type":"agent_message","message":""}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	msgs, err := ParseCodexConversation(path)
	if err != nil {
		t.Fatalf("ParseCodexConversation: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for empty agent message, got %d", len(msgs))
	}
}

// TestParseCodexConversation_UnknownPayloadType tests unknown type filtering.
func TestParseCodexConversation_UnknownPayloadType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	data := `{"timestamp":"2026-01-01T00:00:00Z","type":"event_msg","payload":{"type":"system_log","message":"log entry"}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	msgs, err := ParseCodexConversation(path)
	if err != nil {
		t.Fatalf("ParseCodexConversation: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for unknown type, got %d", len(msgs))
	}
}

// TestParseCodexConversation_NonEventMsgType tests filtering of non-event_msg entries.
func TestParseCodexConversation_NonEventMsgType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	data := `{"timestamp":"2026-01-01T00:00:00Z","type":"init","payload":{"type":"user_message","message":"hi"}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	msgs, err := ParseCodexConversation(path)
	if err != nil {
		t.Fatalf("ParseCodexConversation: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for non-event_msg type, got %d", len(msgs))
	}
}

// TestParseCodexConversation_InvalidJSON tests skipping malformed JSON.
func TestParseCodexConversation_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	data := `not json at all`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	msgs, err := ParseCodexConversation(path)
	if err != nil {
		t.Fatalf("ParseCodexConversation: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for invalid JSON, got %d", len(msgs))
	}
}

// TestParseCodexConversation_InvalidPayloadJSON tests skipping malformed payload.
func TestParseCodexConversation_InvalidPayloadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	data := `{"timestamp":"2026-01-01T00:00:00Z","type":"event_msg","payload":"not json"}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	msgs, err := ParseCodexConversation(path)
	if err != nil {
		t.Fatalf("ParseCodexConversation: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for invalid payload, got %d", len(msgs))
	}
}

// TestParseCodexConversation_NonexistentFile tests error for nonexistent file.
func TestParseCodexConversation_NonexistentFile(t *testing.T) {
	_, err := ParseCodexConversation("/nonexistent/path.jsonl")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// TestParseCodexConversation_UserAndAgentMessages tests full parsing.
func TestParseCodexConversation_UserAndAgentMessages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	lines := []string{
		`{"timestamp":"2026-01-01T00:00:00Z","type":"event_msg","payload":{"type":"user_message","message":"hello"}}`,
		`{"timestamp":"2026-01-01T00:00:01Z","type":"event_msg","payload":{"type":"agent_message","message":"hi there"}}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	msgs, err := ParseCodexConversation(path)
	if err != nil {
		t.Fatalf("ParseCodexConversation: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected role 'user', got %q", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", msgs[1].Role)
	}
}
