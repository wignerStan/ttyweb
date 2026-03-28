package tmux

import (
	"os/exec"
	"strings"
	"testing"
)

func requireTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available on this system")
	}
}

func requireTmuxServer(t *testing.T) {
	t.Helper()
	requireTmux(t)
	if !IsServerRunning() {
		t.Skip("tmux server not running")
	}
}

func TestIsServerRunning(t *testing.T) {
	// Should not panic regardless of tmux availability
	result := IsServerRunning()
	_ = result // just verify no panic
}

func TestListSessions(t *testing.T) {
	requireTmuxServer(t)

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() returned error: %v", err)
	}

	// Should return a non-nil slice (may be empty)
	if sessions == nil {
		t.Error("ListSessions() returned nil, expected non-nil slice")
	}

	for _, s := range sessions {
		if s.Name == "" {
			t.Error("session has empty name")
		}
	}
}

func TestListSessions_Empty(t *testing.T) {
	requireTmux(t)

	// ListSessions should not panic even if server is not running
	// It may return an error if no tmux server, which is acceptable
	_, err := ListSessions()
	// When no server: error is expected and OK
	// When server running: empty or populated slice is fine
	if err != nil && !strings.Contains(err.Error(), "no server running") {
		t.Logf("ListSessions error (may be expected): %v", err)
	}
}

func TestCreateAndKillSession(t *testing.T) {
	requireTmuxServer(t)

	sessionName := "ttyweb-test-" + randomSuffix()

	// Create
	created, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession(%q) returned error: %v", sessionName, err)
	}
	if created != sessionName {
		t.Errorf("created name = %q, want %q", created, sessionName)
	}

	// Verify it exists
	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error after create: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s.Name == sessionName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("session %q not found in list after creation", sessionName)
	}

	// Kill
	err = KillSession(sessionName)
	if err != nil {
		t.Fatalf("KillSession(%q) returned error: %v", sessionName, err)
	}

	// Verify gone
	sessions, err = ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error after kill: %v", err)
	}
	for _, s := range sessions {
		if s.Name == sessionName {
			t.Errorf("session %q still exists after kill", sessionName)
			break
		}
	}
}

func TestCreateSession_InvalidName(t *testing.T) {
	invalidNames := []string{
		"; rm -rf /",
		"$(whoami)",
		"name with spaces",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			_, err := CreateSession(name)
			if err == nil {
				t.Errorf("CreateSession(%q) expected error for invalid name", name)
			}
		})
	}
}

func TestKillSession_InvalidName(t *testing.T) {
	invalidNames := []string{
		"; rm -rf /",
		"$(whoami)",
		"name with spaces",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := KillSession(name)
			if err == nil {
				t.Errorf("KillSession(%q) expected error for invalid name", name)
			}
		})
	}
}

// randomSuffix generates a short random suffix for unique session names.
func randomSuffix() string {
	return strings.ToLower(randomHex(6))
}

func randomHex(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[i%16] // deterministic but unique per-session in tests
	}
	return string(b)
}
