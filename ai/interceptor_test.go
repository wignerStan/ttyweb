package ai

import (
	"testing"
	"unicode/utf8"
)

func TestANSIIntercept_WorkingPattern(t *testing.T) {
	i := NewANSITerminalInterceptor()
	metadata := i.Intercept([]byte("Use Read tool to read file.go\r\n"))
	if len(metadata) == 0 {
		t.Fatal("expected metadata for working pattern")
	}
	if metadata[0].Type != "ai_state_change" {
		t.Errorf("expected type ai_state_change, got %s", metadata[0].Type)
	}
	if metadata[0].Data["state"] != "working" {
		t.Errorf("expected state=working, got %v", metadata[0].Data["state"])
	}
}

func TestANSIIntercept_ApprovalPattern(t *testing.T) {
	i := NewANSITerminalInterceptor()
	metadata := i.Intercept([]byte("\x1b[33m[claude] Need approval\x1b[0m\r\n"))
	if len(metadata) == 0 {
		t.Fatal("expected metadata for approval pattern")
	}
	if metadata[0].Data["state"] != "waiting_approval" {
		t.Errorf("expected state=waiting_approval, got %v", metadata[0].Data["state"])
	}
}

func TestANSIIntercept_ApprovalPatternAlt(t *testing.T) {
	i := NewANSITerminalInterceptor()
	metadata := i.Intercept([]byte("Allow BashTool to execute? [y/n]\r\n"))
	if len(metadata) == 0 {
		t.Fatal("expected metadata for approval pattern")
	}
	if metadata[0].Data["state"] != "waiting_approval" {
		t.Errorf("expected state=waiting_approval, got %v", metadata[0].Data["state"])
	}
}

func TestANSIIntercept_NoMatch(t *testing.T) {
	i := NewANSITerminalInterceptor()
	metadata := i.Intercept([]byte("ls -la\r\n"))
	if len(metadata) != 0 {
		t.Errorf("expected no metadata for non-AI output, got %d", len(metadata))
	}
}

func TestANSIIntercept_MultipleMatchesFirstWins(t *testing.T) {
	i := NewANSITerminalInterceptor()
	metadata := i.Intercept([]byte("Use BashTool to run: Allow BashTool to execute\r\n"))
	if len(metadata) != 1 {
		t.Errorf("expected exactly 1 metadata (first match wins), got %d", len(metadata))
	}
}

func TestANSIIntercept_EmptyInput(t *testing.T) {
	i := NewANSITerminalInterceptor()
	metadata := i.Intercept([]byte{})
	if len(metadata) != 0 {
		t.Error("expected no metadata for empty input")
	}
}

func TestANSIIntercept_CapturesUserInputOnWorking(t *testing.T) {
	i := NewANSITerminalInterceptor()

	// First: trigger working state
	metadata1 := i.Intercept([]byte("Use Read tool to read main.go\r\n"))
	foundWorking := false
	for _, m := range metadata1 {
		if m.Type == "ai_state_change" && m.Data["state"] == "working" {
			foundWorking = true
		}
	}
	if !foundWorking {
		t.Fatal("expected working state")
	}

	// Next: terminal output that looks like user input
	metadata2 := i.Intercept([]byte("fix the authentication bug in login handler\r\n"))
	foundRename := false
	for _, m := range metadata2 {
		if m.Type == "tab_rename" {
			foundRename = true
			summary, ok := m.Data["summary"].(string)
			if !ok || summary == "" {
				t.Error("expected non-empty summary in tab_rename")
			}
			if utf8.RuneCountInString(summary) > 64 {
				t.Errorf("summary should be truncated to 64 chars, got %d", utf8.RuneCountInString(summary))
			}
		}
	}
	if !foundRename {
		t.Error("expected tab_rename metadata when in working state")
	}
}

func TestANSIIntercept_NoTabRenameWhenIdle(t *testing.T) {
	i := NewANSITerminalInterceptor()

	// Output that looks like user input, but no working state has been triggered
	metadata := i.Intercept([]byte("fix the authentication bug\r\n"))
	for _, m := range metadata {
		if m.Type == "tab_rename" {
			t.Error("expected no tab_rename when not in working state")
		}
	}
}

func TestANSIIntercept_NoTabRenameForToolOutput(t *testing.T) {
	i := NewANSITerminalInterceptor()

	// Trigger working state
	i.Intercept([]byte("Use Read tool to read main.go\r\n"))

	// Output that looks like tool execution, not user input
	metadata := i.Intercept([]byte("Reading file config/settings.json\r\n"))
	for _, m := range metadata {
		if m.Type == "tab_rename" {
			t.Error("expected no tab_rename for tool-like output")
		}
	}
}

func TestANSIIntercept_TabRenameClearedOnIdle(t *testing.T) {
	i := NewANSITerminalInterceptor()

	// Trigger working, then user input, then idle
	i.Intercept([]byte("Use Read tool to read main.go\r\n"))
	i.Intercept([]byte("fix the authentication bug\r\n"))
	i.Intercept([]byte("Human:\r\n"))

	// Now in idle state — user input should not produce tab_rename
	metadata := i.Intercept([]byte("do something else\r\n"))
	for _, m := range metadata {
		if m.Type == "tab_rename" {
			t.Error("expected no tab_rename after idle state")
		}
	}
}

func TestLooksLikeToolOutput_True(t *testing.T) {
	cases := []string{
		"Use Read tool to read file",
		"Applying patch to main.go",
		"Reading config.json",
		"Editing source file",
		"Writing new test file",
		"[INFO] Build succeeded",
		"# This is a comment",
		"$ git status",
	}
	for _, c := range cases {
		if !looksLikeToolOutput(c) {
			t.Errorf("expected looksLikeToolOutput(%q) = true", c)
		}
	}
}

func TestLooksLikeToolOutput_False(t *testing.T) {
	cases := []string{
		"fix the authentication bug",
		"add a new feature",
		"hello world",
		"please review my code",
		"run the tests",
	}
	for _, c := range cases {
		if looksLikeToolOutput(c) {
			t.Errorf("expected looksLikeToolOutput(%q) = false", c)
		}
	}
}

func TestTruncate(t *testing.T) {
	// No truncation needed
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	// Exact length
	if got := truncate("hello", 5); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	// Truncation with ellipsis
	if got := truncate("hello world", 5); got != "hello\u2026" {
		t.Errorf("expected 'hello...', got %q", got)
	}
	// Empty string
	if got := truncate("", 5); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestANSIIntercept_TabRenameTruncation(t *testing.T) {
	i := NewANSITerminalInterceptor()

	// Trigger working state
	i.Intercept([]byte("Use Read tool to read main.go\r\n"))

	// Long input exceeding 64 chars
	longInput := "this is a very long user input that should definitely be truncated because it exceeds the maximum allowed length of sixty-four characters"
	metadata := i.Intercept([]byte(longInput + "\r\n"))
	foundRename := false
	for _, m := range metadata {
		if m.Type == "tab_rename" {
			foundRename = true
			summary, _ := m.Data["summary"].(string)
			if utf8.RuneCountInString(summary) > 65 {
				t.Errorf("summary should be <= 65 runes (64 + ellipsis), got %d: %q", utf8.RuneCountInString(summary), summary)
			}
			runes := []rune(longInput)
			expected := string(runes[:64]) + "\u2026"
			if summary != expected {
				t.Errorf("expected truncated summary with ellipsis, got %q", summary)
			}
		}
	}
	if !foundRename {
		t.Error("expected tab_rename for long input")
	}
}
