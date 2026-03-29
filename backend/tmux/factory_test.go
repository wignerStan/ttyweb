package tmux

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

func TestWithCloseSignal(t *testing.T) {
	t.Parallel()
	opt := WithCloseSignal(9) // SIGKILL
	s := &TmuxSlave{}
	opt(s)
	if s.closeSignal != 9 {
		t.Fatalf("expected signal 9, got %d", s.closeSignal)
	}
}

func TestWithCloseTimeout(t *testing.T) {
	t.Parallel()
	opt := WithCloseTimeout(5)
	s := &TmuxSlave{}
	opt(s)
	if s.closeTimeout != 5 {
		t.Fatalf("expected timeout 5, got %d", s.closeTimeout)
	}
}
