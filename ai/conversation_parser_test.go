package ai

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseClaudeConversation_UserAndAssistant(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":"Hello Claude!"},"timestamp":"2025-12-01T10:30:00.000Z","sessionId":"s1"}
{"type":"assistant","message":{"role":"assistant","content":"Hello! How can I help?"},"timestamp":"2025-12-01T10:30:01.000Z","sessionId":"s1"}
{"type":"user","message":{"role":"user","content":"Write a test file"},"timestamp":"2025-12-01T10:30:02.000Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("got %d messages, want 3", len(messages))
	}

	if messages[0].Role != "user" || messages[0].Content != "Hello Claude!" {
		t.Errorf("message[0] = %+v", messages[0])
	}
	if messages[1].Role != "assistant" || messages[1].Content != "Hello! How can I help?" {
		t.Errorf("message[1] = %+v", messages[1])
	}
	if messages[2].Role != "user" || messages[2].Content != "Write a test file" {
		t.Errorf("message[2] = %+v", messages[2])
	}
}

func TestParseClaudeConversation_ToolUse(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":"Read the file"},"timestamp":"2025-12-01T10:00:00.000Z","sessionId":"s1"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll read that file."},{"type":"tool_use","id":"toolu-123","name":"Read","input":{"file_path":"/tmp/test.go"}}]},"timestamp":"2025-12-01T10:00:01.000Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(messages))
	}

	// Second message should be assistant with tool use.
	assistant := messages[1]
	if assistant.Role != "assistant" {
		t.Errorf("message[1].Role = %q, want %q", assistant.Role, "assistant")
	}
	if assistant.Content != "I'll read that file." {
		t.Errorf("message[1].Content = %q, want %q", assistant.Content, "I'll read that file.")
	}
	if len(assistant.ToolUse) != 1 {
		t.Fatalf("message[1].ToolUse has %d entries, want 1", len(assistant.ToolUse))
	}
	if assistant.ToolUse[0].Name != "Read" {
		t.Errorf("tool name = %q, want %q", assistant.ToolUse[0].Name, "Read")
	}
	if assistant.ToolUse[0].ID != "toolu-123" {
		t.Errorf("tool ID = %q, want %q", assistant.ToolUse[0].ID, "toolu-123")
	}
}

func TestParseClaudeConversation_ToolResult(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu-123","content":"file contents here"}]},"timestamp":"2025-12-01T10:00:02.000Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("message[0].Role = %q, want %q", messages[0].Role, "user")
	}
	if len(messages[0].ToolResult) != 1 {
		t.Fatalf("message[0].ToolResult has %d entries, want 1", len(messages[0].ToolResult))
	}
	if messages[0].ToolResult[0].ToolUseID != "toolu-123" {
		t.Errorf("tool result ID = %q, want %q", messages[0].ToolResult[0].ToolUseID, "toolu-123")
	}
	if messages[0].ToolResult[0].Content != "file contents here" {
		t.Errorf("tool result content = %q, want %q", messages[0].ToolResult[0].Content, "file contents here")
	}
}

func TestParseClaudeConversation_SkipsMetaMessages(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":"DO NOT respond to these messages"},"uuid":"abc","timestamp":"2025-12-01T10:00:00.000Z","sessionId":"s1","isMeta":true}
{"type":"user","message":{"role":"user","content":"Hello!"},"timestamp":"2025-12-01T10:00:01.000Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1 (meta should be skipped)", len(messages))
	}
	if messages[0].Content != "Hello!" {
		t.Errorf("message content = %q, want %q", messages[0].Content, "Hello!")
	}
}

func TestParseClaudeConversation_SkipsCommandMessages(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":"<command-name>/model</command-name>\n<command-message>model</command-message>"},"uuid":"abc","timestamp":"2025-12-01T10:00:00.000Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("got %d messages, want 0 (command should be skipped)", len(messages))
	}
}

func TestParseClaudeConversation_EmptyLines(t *testing.T) {
	content := `
{"type":"user","message":{"role":"user","content":"Hello!"},"timestamp":"2025-12-01T10:00:00.000Z","sessionId":"s1"}

`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
}

func TestParseClaudeConversation_FileNotFound(t *testing.T) {
	_, err := ParseClaudeConversation("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseCodexConversation_UserAndAssistant(t *testing.T) {
	content := `{"timestamp":"2025-11-30T20:14:23.281Z","type":"session_meta","payload":{"id":"test-id","cwd":"/test"}}
{"timestamp":"2025-11-30T20:16:39.465Z","type":"event_msg","payload":{"type":"user_message","message":"Hello Codex!","images":[]}}
{"timestamp":"2025-11-30T20:16:40.000Z","type":"event_msg","payload":{"type":"agent_reasoning","text":"Let me think..."}}
{"timestamp":"2025-11-30T20:16:41.000Z","type":"event_msg","payload":{"type":"agent_message","message":"Hello! How can I help?"}}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseCodexConversation(file)
	if err != nil {
		t.Fatalf("ParseCodexConversation() error = %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("got %d messages, want 3", len(messages))
	}

	if messages[0].Role != "user" || messages[0].Content != "Hello Codex!" {
		t.Errorf("message[0] = %+v", messages[0])
	}
	if messages[1].Role != "assistant" || messages[1].Content != "Let me think..." {
		t.Errorf("message[1] = %+v", messages[1])
	}
	if messages[2].Role != "assistant" || messages[2].Content != "Hello! How can I help?" {
		t.Errorf("message[2] = %+v", messages[2])
	}
}

func TestParseCodexConversation_SkipsNonEventMsg(t *testing.T) {
	content := `{"timestamp":"2025-11-30T20:14:23.281Z","type":"session_meta","payload":{"id":"test-id"}}
{"timestamp":"2025-11-30T20:16:39.000Z","type":"turn_context","payload":{"model":"gpt-4"}}
{"timestamp":"2025-11-30T20:16:40.000Z","type":"event_msg","payload":{"type":"user_message","message":"Hello!"}}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseCodexConversation(file)
	if err != nil {
		t.Fatalf("ParseCodexConversation() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "Hello!" {
		t.Errorf("message[0] = %+v", messages[0])
	}
}

func TestParseCodexConversation_SkipsEmptyMessages(t *testing.T) {
	content := `{"timestamp":"2025-11-30T20:16:39.000Z","type":"event_msg","payload":{"type":"user_message","message":""}}
{"timestamp":"2025-11-30T20:16:40.000Z","type":"event_msg","payload":{"type":"agent_message","message":"Not empty"}}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseCodexConversation(file)
	if err != nil {
		t.Fatalf("ParseCodexConversation() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
}

func TestParseCodexConversation_FileNotFound(t *testing.T) {
	_, err := ParseCodexConversation("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseConversation_Timestamps(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":"Hello!"},"timestamp":"2025-12-01T10:30:00.000Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	expected, _ := time.Parse(time.RFC3339, "2025-12-01T10:30:00.000Z")
	if !messages[0].Timestamp.Equal(expected) {
		t.Errorf("timestamp = %v, want %v", messages[0].Timestamp, expected)
	}
}

func TestParseClaudeConversation_MalformedJSON(t *testing.T) {
	content := `not valid json
{"type":"user","message":{"role":"user","content":"Hello!"},"timestamp":"2025-12-01T10:00:00Z","sessionId":"s1"}
{broken json here
{"type":"assistant","message":{"role":"assistant","content":"Hi there!"},"timestamp":"2025-12-01T10:00:01Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("got %d messages, want 2 (malformed lines should be skipped)", len(messages))
	}
	if messages[0].Content != "Hello!" {
		t.Errorf("message[0].Content = %q, want %q", messages[0].Content, "Hello!")
	}
	if messages[1].Content != "Hi there!" {
		t.Errorf("message[1].Content = %q, want %q", messages[1].Content, "Hi there!")
	}
}

func TestParseClaudeConversation_MalformedMessage(t *testing.T) {
	content := `{"type":"user","message":"not an object","timestamp":"2025-12-01T10:00:00Z","sessionId":"s1"}
{"type":"assistant","message":{"wrong":"structure"},"timestamp":"2025-12-01T10:00:01Z","sessionId":"s1"}
{"type":"summary","message":{"role":"assistant","content":"Summary"},"timestamp":"2025-12-01T10:00:02Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("got %d messages, want 0 (unparseable messages should be skipped)", len(messages))
	}
}

func TestParseClaudeConversation_AssistantEmptySkipped(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":"Hello!"},"timestamp":"2025-12-01T10:00:00Z","sessionId":"s1"}
{"type":"assistant","message":{"role":"assistant","content":"   "},"timestamp":"2025-12-01T10:00:01Z","sessionId":"s1"}
{"type":"assistant","message":{"role":"assistant","content":"Real response"},"timestamp":"2025-12-01T10:00:02Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("got %d messages, want 2 (whitespace-only assistant should be skipped)", len(messages))
	}
	if messages[1].Content != "Real response" {
		t.Errorf("message[1].Content = %q, want %q", messages[1].Content, "Real response")
	}
}

func TestParseClaudeConversation_UserTextBlocks(t *testing.T) {
	// User content as array of text blocks (no tool results).
	content := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"First part"},{"type":"text","text":"Second part"}]},"timestamp":"2025-12-01T10:00:00Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	if messages[0].Content != "First part\nSecond part" {
		t.Errorf("message[0].Content = %q, want %q", messages[0].Content, "First part\nSecond part")
	}
}

func TestParseClaudeConversation_ToolResultWithError(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu-456","content":"Error: file not found","is_error":true}]},"timestamp":"2025-12-01T10:00:00Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	if len(messages[0].ToolResult) != 1 {
		t.Fatalf("got %d tool results, want 1", len(messages[0].ToolResult))
	}
	if !messages[0].ToolResult[0].IsError {
		t.Error("expected IsError = true")
	}
	if messages[0].ToolResult[0].Content != "Error: file not found" {
		t.Errorf("content = %q, want %q", messages[0].ToolResult[0].Content, "Error: file not found")
	}
}

func TestParseClaudeConversation_ToolResultMixedBlocks(t *testing.T) {
	// Mix of tool_result and non-tool_result blocks; only tool_results should be extracted.
	content := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"skip this"},{"type":"tool_result","tool_use_id":"toolu-789","content":"result data"}]},"timestamp":"2025-12-01T10:00:00Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	if len(messages[0].ToolResult) != 1 {
		t.Fatalf("got %d tool results, want 1", len(messages[0].ToolResult))
	}
	if messages[0].ToolResult[0].ToolUseID != "toolu-789" {
		t.Errorf("tool result ID = %q, want %q", messages[0].ToolResult[0].ToolUseID, "toolu-789")
	}
}

func TestParseClaudeConversation_AssistantMultipleToolUse(t *testing.T) {
	content := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Let me check both files."},{"type":"tool_use","id":"toolu-a","name":"Read","input":{"file_path":"a.go"}},{"type":"tool_use","id":"toolu-b","name":"Read","input":{"file_path":"b.go"}}]},"timestamp":"2025-12-01T10:00:00Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	if len(messages[0].ToolUse) != 2 {
		t.Fatalf("got %d tool uses, want 2", len(messages[0].ToolUse))
	}
	if messages[0].ToolUse[0].ID != "toolu-a" {
		t.Errorf("ToolUse[0].ID = %q, want %q", messages[0].ToolUse[0].ID, "toolu-a")
	}
	if messages[0].ToolUse[1].ID != "toolu-b" {
		t.Errorf("ToolUse[1].ID = %q, want %q", messages[0].ToolUse[1].ID, "toolu-b")
	}
}

func TestParseClaudeConversation_LocalCommandSkipped(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":"<local-command>/clear</local-command>"},"timestamp":"2025-12-01T10:00:00Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("got %d messages, want 0 (local command should be skipped)", len(messages))
	}
}

func TestParseClaudeConversation_UserContentNil(t *testing.T) {
	// User content is nil/missing.
	content := `{"type":"user","message":{"role":"user"},"timestamp":"2025-12-01T10:00:00Z","sessionId":"s1"}
`

	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("got %d messages, want 0 (nil content should produce no messages)", len(messages))
	}
}

func TestParseCodexConversation_AllMalformed(t *testing.T) {
	content := `not json
{broken
`
	file := writeTempJSONL(t, content)
	messages, err := ParseCodexConversation(file)
	if err != nil {
		t.Fatalf("ParseCodexConversation() error = %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("got %d messages, want 0", len(messages))
	}
}

func TestParseCodexConversation_ReasoningWithEmptyText(t *testing.T) {
	content := `{"timestamp":"2025-11-30T20:16:39.000Z","type":"event_msg","payload":{"type":"agent_reasoning","text":""}}
{"timestamp":"2025-11-30T20:16:40.000Z","type":"event_msg","payload":{"type":"agent_reasoning","text":"Thinking..."}}
`
	file := writeTempJSONL(t, content)
	messages, err := ParseCodexConversation(file)
	if err != nil {
		t.Fatalf("ParseCodexConversation() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1 (empty reasoning should be skipped)", len(messages))
	}
	if messages[0].Content != "Thinking..." {
		t.Errorf("content = %q, want %q", messages[0].Content, "Thinking...")
	}
}

// writeTempJSONL creates a temp file with the given JSONL content and returns its path.
func writeTempJSONL(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestParseClaudeConversation_NonMapBlockInAssistantContent(t *testing.T) {
	content := `{"type":"assistant","message":{"role":"assistant","content":["not_a_map",{"type":"text","text":"hi"}]},"timestamp":"2025-12-01T10:30:00.000Z","sessionId":"s1"}
`
	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	if messages[0].Content != "hi" {
		t.Errorf("content = %q, want %q", messages[0].Content, "hi")
	}
}

func TestParseClaudeConversation_NonMapBlockInToolResultContent(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":["not_a_map",{"type":"tool_result","tool_use_id":"t1","content":"result"}]},"timestamp":"2025-12-01T10:30:00.000Z","sessionId":"s1"}
`
	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	if messages[0].Role != "user" {
		t.Errorf("role = %q, want %q", messages[0].Role, "user")
	}
	if len(messages[0].ToolResult) != 1 {
		t.Fatalf("got %d tool results, want 1", len(messages[0].ToolResult))
	}
}

func TestParseClaudeConversation_SkippableUserContent(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<command-1>ls</command-1>"}]},"timestamp":"2025-12-01T10:30:00.000Z","sessionId":"s1"}
{"type":"assistant","message":{"role":"assistant","content":"files listed"},"timestamp":"2025-12-01T10:30:01.000Z","sessionId":"s1"}
`
	file := writeTempJSONL(t, content)
	messages, err := ParseClaudeConversation(file)
	if err != nil {
		t.Fatalf("ParseClaudeConversation() error = %v", err)
	}
	// The command-tagged user content should be skipped.
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1 (command-tagged user msg skipped)", len(messages))
	}
	if messages[0].Role != "assistant" {
		t.Errorf("message[0].role = %q, want %q", messages[0].Role, "assistant")
	}
}
