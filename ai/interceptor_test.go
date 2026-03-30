package ai

import (
	"testing"
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
