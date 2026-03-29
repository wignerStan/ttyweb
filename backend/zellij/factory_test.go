package zellij

import (
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
