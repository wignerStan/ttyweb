package zellij

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
	if f.Name() != "zellij" {
		t.Fatalf("expected 'zellij', got %q", f.Name())
	}
}

func TestFactoryIsAvailable(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	// IsAvailable checks if zellij server is running.
	// We just verify it doesn't panic.
	_ = f.IsAvailable()
}

func TestFactoryListSessions(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	// ListSessions calls zellij which may not be running.
	// We just verify it doesn't panic.
	_, _ = f.ListSessions()
}

func TestFactoryCreateSession(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	// CreateSession requires a running zellij server.
	// We just verify it doesn't panic.
	_, _ = f.CreateSession("test-session")
}

func TestFactoryGetSessionDetail(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	// GetSessionDetail parses list output.
	// We just verify it doesn't panic.
	_, _ = f.GetSessionDetail("nonexistent")
}

func TestFactoryKillSession(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	// KillSession requires a running zellij server.
	// We just verify it doesn't panic.
	_ = f.KillSession("nonexistent")
}

func TestFactoryNew_WithSession(t *testing.T) {
	t.Parallel()
	f := NewFactory("default-session")
	params := map[string][]string{"session": {"my-session"}}
	slave, err := f.New(params, nil)
	if err != nil {
		t.Logf("New with session param failed (expected if zellij unavailable): %v", err)
	} else {
		defer slave.Close()
		t.Log("New with session param succeeded")
	}
}

func TestFactoryNew_DefaultSession(t *testing.T) {
	t.Parallel()
	f := NewFactory("fallback-session")
	slave, err := f.New(nil, nil)
	if err != nil {
		t.Logf("New with default session failed (expected if zellij unavailable): %v", err)
	} else {
		defer slave.Close()
		t.Log("New with default session succeeded")
	}
}

func TestFactoryListSessions_Marshal(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	data, err := f.ListSessions()
	if err != nil {
		t.Logf("ListSessions error (expected if zellij unavailable): %v", err)
		return
	}
	// Verify the result is valid JSON
	if !json.Valid(data) {
		t.Errorf("ListSessions returned invalid JSON: %s", string(data))
	}
	// Verify it unmarshals to a valid JSON structure
	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		t.Errorf("ListSessions JSON unmarshal failed: %v", err)
	}
}

func TestFactoryCreateSession_Invalid(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	_, err := f.CreateSession("bad;name")
	if err == nil {
		t.Error("CreateSession with invalid name should return error")
	}
}

func TestFactoryGetSessionDetail_NotFound(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	_, err := f.GetSessionDetail("nonexistent-session-xyz")
	if err == nil {
		t.Error("GetSessionDetail for nonexistent session should return error")
	}
}

func TestFactoryKillSession_Invalid(t *testing.T) {
	t.Parallel()
	f := NewFactory("s1")
	err := f.KillSession("bad;name")
	if err == nil {
		t.Error("KillSession with invalid name should return error")
	}
}
