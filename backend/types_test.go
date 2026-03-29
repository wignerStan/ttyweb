package backend

import (
	"errors"
	"testing"
)

func TestNoSessionManagerIsAvailable(t *testing.T) {
	t.Parallel()
	mgr := NoSessionManager{}
	if mgr.IsAvailable() {
		t.Error("NoSessionManager.IsAvailable() should return false")
	}
}

func TestNoSessionManagerListSessions(t *testing.T) {
	t.Parallel()
	mgr := NoSessionManager{}
	raw, err := mgr.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() unexpected error: %v", err)
	}
	if raw != nil {
		t.Errorf("ListSessions() raw = %v, want nil", raw)
	}
}

func TestNoSessionManagerCreateSession(t *testing.T) {
	t.Parallel()
	mgr := NoSessionManager{}
	name, err := mgr.CreateSession("test-session")
	if err == nil {
		t.Fatal("CreateSession() expected error, got nil")
	}
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("CreateSession() error = %v, want ErrNotSupported", err)
	}
	if name != "" {
		t.Errorf("CreateSession() name = %q, want empty string", name)
	}
}

func TestNoSessionManagerCreateSessionWithCommand(t *testing.T) {
	t.Parallel()
	mgr := NoSessionManager{}
	name, err := mgr.CreateSession("test", "vim", "-O")
	if err == nil {
		t.Fatal("CreateSession() with command expected error, got nil")
	}
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("CreateSession() error = %v, want ErrNotSupported", err)
	}
	if name != "" {
		t.Errorf("CreateSession() name = %q, want empty string", name)
	}
}

func TestNoSessionManagerGetSessionDetail(t *testing.T) {
	t.Parallel()
	mgr := NoSessionManager{}
	raw, err := mgr.GetSessionDetail("some-session")
	if err == nil {
		t.Fatal("GetSessionDetail() expected error, got nil")
	}
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("GetSessionDetail() error = %v, want ErrNotSupported", err)
	}
	if raw != nil {
		t.Errorf("GetSessionDetail() raw = %v, want nil", raw)
	}
}

func TestNoSessionManagerKillSession(t *testing.T) {
	t.Parallel()
	mgr := NoSessionManager{}
	err := mgr.KillSession("some-session")
	if err == nil {
		t.Fatal("KillSession() expected error, got nil")
	}
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("KillSession() error = %v, want ErrNotSupported", err)
	}
}

func TestNoSessionManagerKillSessionEmpty(t *testing.T) {
	t.Parallel()
	mgr := NoSessionManager{}
	err := mgr.KillSession("")
	if err == nil {
		t.Fatal("KillSession(\"\") expected error, got nil")
	}
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("KillSession(\"\") error = %v, want ErrNotSupported", err)
	}
}

func TestNoSessionManagerImplementsSessionManager(t *testing.T) {
	t.Parallel()
	// Compile-time interface check.
	var _ SessionManager = NoSessionManager{}
}

func TestErrNotSupportedMessage(t *testing.T) {
	t.Parallel()
	expected := "operation not supported by this backend"
	if ErrNotSupported.Error() != expected {
		t.Errorf("ErrNotSupported.Error() = %q, want %q", ErrNotSupported.Error(), expected)
	}
}

func TestNoSessionManagerEmptyName(t *testing.T) {
	t.Parallel()
	mgr := NoSessionManager{}
	name, err := mgr.CreateSession("")
	if err == nil {
		t.Fatal("CreateSession(\"\") expected error, got nil")
	}
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("CreateSession(\"\") error = %v, want ErrNotSupported", err)
	}
	if name != "" {
		t.Errorf("CreateSession(\"\") name = %q, want empty string", name)
	}
}
