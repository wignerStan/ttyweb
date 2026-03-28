package zellij

import (
	"os/exec"
	"strings"
	"testing"
)

func requireZellij(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("zellij"); err != nil {
		t.Skip("zellij not available on this system")
	}
}

func TestIsServerRunning(t *testing.T) {
	// Should not panic regardless of zellij availability
	result := IsServerRunning()
	_ = result // just verify no panic
}

func TestListSessions(t *testing.T) {
	requireZellij(t)

	// Should not panic; may return empty or error if no server
	sessions, err := ListSessions()
	if err != nil {
		t.Logf("ListSessions error (may be expected if no server): %v", err)
		return
	}

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
	requireZellij(t)

	// Without a running server, sessions should be nil or empty
	sessions, err := ListSessions()
	if err != nil {
		t.Logf("ListSessions error (expected when no server): %v", err)
		return
	}
	if len(sessions) > 0 {
		t.Logf("unexpected sessions found (non-empty environment): %d", len(sessions))
	}
}

func TestCreateAndKillSession(t *testing.T) {
	requireZellij(t)

	sessionName := "ttyweb-test-" + randomSuffix()

	// Create
	created, err := CreateSession(sessionName)
	if err != nil {
		t.Skipf("skipping: zellij could not create session (no session manager running?): %v", err)
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

	// Verify gone (with tolerance for race)
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

// randomSuffix generates a deterministic short suffix for unique session names.
func randomSuffix() string {
	return strings.ToLower(randomHex(6))
}

func randomHex(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[i%16]
	}
	return string(b)
}
