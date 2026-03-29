package zellij

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
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

func TestListSessions_Parse(t *testing.T) {
	requireZellij(t)

	// Verify ListSessions returns parseable data without panicking
	sessions, err := ListSessions()
	if err != nil {
		t.Logf("ListSessions error (expected if no server): %v", err)
		return
	}
	if sessions == nil {
		t.Log("no sessions found")
	}
	// Verify all session names are non-empty
	for _, s := range sessions {
		if s.Name == "" {
			t.Error("session has empty name")
		}
	}
}

func TestSessionsJSON(t *testing.T) {
	requireZellij(t)

	data, err := SessionsJSON()
	if err != nil {
		t.Logf("SessionsJSON error (expected if no server): %v", err)
		return
	}
	if !json.Valid(data) {
		t.Errorf("SessionsJSON returned invalid JSON: %s", string(data))
	}
	// Verify it unmarshals to a valid slice of sessions
	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		t.Errorf("SessionsJSON unmarshal failed: %v", err)
	}
}

func TestZellijOutput_Invalid(t *testing.T) {
	requireZellij(t)

	_, err := zellijOutput("nonexistent-subcommand-xyz")
	if err == nil {
		t.Error("zellijOutput with invalid subcommand should return error")
	}
}

func TestZellijExec_Invalid(t *testing.T) {
	requireZellij(t)

	_, err := zellijExec("nonexistent-subcommand-xyz")
	if err == nil {
		t.Error("zellijExec with invalid subcommand should return error")
	}
}

func TestCreateSession_WithCommand(t *testing.T) {
	requireZellij(t)

	sessionName := "ttyweb-cmd-" + randomSuffix()
	created, err := CreateSession(sessionName, "true")
	if err != nil {
		t.Skipf("skipping: zellij could not create session with command: %v", err)
	}
	if created != sessionName {
		t.Errorf("created name = %q, want %q", created, sessionName)
	}
	// Clean up
	_ = KillSession(sessionName)
}

// createRawSession starts a zellij session directly (bypassing CreateSession)
// to exercise ListSessions parsing paths.
func createRawSession(t *testing.T, name string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "zellij", "-s", name)
	if err := cmd.Start(); err != nil {
		t.Skipf("skipping: could not start zellij session: %v", err)
	}
	// Give zellij time to register the session
	time.Sleep(500 * time.Millisecond)
	t.Cleanup(func() {
		kill := exec.CommandContext(context.Background(), "zellij", "kill-session", name)
		_ = kill.Run()
	})
}

func TestListSessions_WithSession(t *testing.T) {
	requireZellij(t)

	sessionName := "ttyweb-list-" + randomSuffix()
	createRawSession(t, sessionName)

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("ListSessions() returned empty slice, expected at least one session")
	}

	found := false
	for _, s := range sessions {
		if s.Name != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one session with non-empty name")
	}
}

func TestGetSessionDetail_Found(t *testing.T) {
	requireZellij(t)

	sessionName := "ttyweb-detail-" + randomSuffix()
	createRawSession(t, sessionName)

	// First, get the actual name from ListSessions (may contain ANSI codes)
	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("ListSessions() returned empty, cannot test GetSessionDetail")
	}

	// Find our session in the list
	var actualName string
	for _, s := range sessions {
		if strings.Contains(s.Name, sessionName) {
			actualName = s.Name
			break
		}
	}
	if actualName == "" {
		t.Fatalf("session %q not found in list", sessionName)
	}

	f := NewFactory("s1")
	data, err := f.GetSessionDetail(actualName)
	if err != nil {
		t.Fatalf("GetSessionDetail(%q) error: %v", actualName, err)
	}
	if !json.Valid(data) {
		t.Errorf("GetSessionDetail returned invalid JSON: %s", string(data))
	}
	var detail SessionDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		t.Errorf("GetSessionDetail JSON unmarshal failed: %v", err)
	}
}

func TestSessionsJSON_WithSession(t *testing.T) {
	requireZellij(t)

	sessionName := "ttyweb-json-" + randomSuffix()
	createRawSession(t, sessionName)

	data, err := SessionsJSON()
	if err != nil {
		t.Fatalf("SessionsJSON() error: %v", err)
	}
	if !json.Valid(data) {
		t.Errorf("SessionsJSON returned invalid JSON: %s", string(data))
	}
	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		t.Errorf("SessionsJSON unmarshal failed: %v", err)
	}
	if len(sessions) == 0 {
		t.Error("expected at least one session")
	}
}

func TestFactoryListSessions_WithSession(t *testing.T) {
	requireZellij(t)

	sessionName := "ttyweb-factory-" + randomSuffix()
	createRawSession(t, sessionName)

	f := NewFactory("s1")
	data, err := f.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if !json.Valid(data) {
		t.Errorf("ListSessions returned invalid JSON: %s", string(data))
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
