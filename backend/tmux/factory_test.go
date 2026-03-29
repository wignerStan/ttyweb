package tmux

import (
	"encoding/json"
	"testing"
)

func TestNewFactory(t *testing.T) {
	t.Parallel()
	f := NewFactory("default-session")
	if f == nil {
		t.Fatal("expected non-nil factory")
	}
	if f.defaultSession != "default-session" {
		t.Fatalf("expected default-session, got %q", f.defaultSession)
	}
}

func TestFactoryName(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	if f.Name() != "tmux" {
		t.Fatalf("expected 'tmux', got %q", f.Name())
	}
}

func TestFactoryIsAvailable(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	// IsAvailable checks if tmux server is running.
	// We just verify it doesn't panic.
	_ = f.IsAvailable()
}

func TestFactoryListSessions(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	// ListSessions calls tmux which may not be running.
	// We just verify it doesn't panic.
	_, _ = f.ListSessions()
}

func TestFactoryCreateSession(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	// CreateSession requires a running tmux server.
	// We just verify it doesn't panic.
	_, _ = f.CreateSession("test-session")
}

func TestFactoryGetSessionDetail(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	// GetSessionDetail requires a running tmux server.
	// We just verify it doesn't panic.
	_, _ = f.GetSessionDetail("nonexistent")
}

func TestFactoryKillSession(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	// KillSession requires a running tmux server.
	// We just verify it doesn't panic.
	_ = f.KillSession("nonexistent")
}

func TestFactoryNew_WithSession(t *testing.T) {
	t.Parallel()
	f := NewFactory("default-sess")
	params := map[string][]string{"session": {"my-session"}}
	_, err := f.New(params, nil)
	// NewTmuxSlave will fail because no tmux server is available or session doesn't exist
	if err == nil {
		// If it somehow succeeded (tmux running), that's fine too
		t.Log("New succeeded (tmux available)")
	}
}

func TestFactoryNew_DefaultSession(t *testing.T) {
	t.Parallel()
	f := NewFactory("default-sess")
	_, err := f.New(nil, nil)
	if err == nil {
		t.Log("New succeeded (tmux available)")
	}
}

func TestFactoryNew_WithPane(t *testing.T) {
	t.Parallel()
	f := NewFactory("default-sess")
	params := map[string][]string{
		"session": {"my-session"},
		"pane":    {"%0"},
	}
	_, err := f.New(params, nil)
	if err == nil {
		t.Log("New succeeded (tmux available)")
	}
}

func TestFactoryListSessions_Marshal(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	data, err := f.ListSessions()
	// Error is OK if tmux isn't running
	if err != nil {
		return
	}
	// If no error, verify it's valid JSON
	var sessions []json.RawMessage
	if err := json.Unmarshal(data, &sessions); err != nil {
		t.Fatalf("ListSessions returned invalid JSON: %v\n%s", err, string(data))
	}
}

func TestFactoryCreateSession_Invalid(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	_, err := f.CreateSession("bad;name")
	if err == nil {
		t.Error("CreateSession(bad;name) expected error for invalid name")
	}
}

func TestFactoryGetSessionDetail_NotFound(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	_, err := f.GetSessionDetail("nonexistent-session-xyz")
	// Either validation error (no tmux) or session-not-found error
	if err == nil {
		t.Error("GetSessionDetail(nonexistent) expected error")
	}
}

func TestFactoryKillSession_Invalid(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	err := f.KillSession("bad;name")
	if err == nil {
		t.Error("KillSession(bad;name) expected error for invalid name")
	}
}
