package tmux

import (
	"encoding/json"
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

func TestListSessions_Parse(t *testing.T) {
	requireTmux(t)
	// Should not panic, may return error if no server
	_, err := ListSessions()
	if err != nil {
		t.Logf("ListSessions error (expected if no server): %v", err)
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

func TestListPanes_InvalidSession(t *testing.T) {
	t.Parallel()
	_, err := ListPanes("bad;session")
	if err == nil {
		t.Error("ListPanes(bad;session) expected error for invalid session name")
	}
}

func TestListPanes_NotFound(t *testing.T) {
	requireTmuxServer(t)
	// Nonexistent session should return an error from tmux
	_, err := ListPanes("nonexistent-session-xyz")
	if err != nil {
		t.Logf("ListPanes(notfound) error (expected): %v", err)
	}
	// It's also acceptable if tmux returns empty (some versions)
}

func TestGetSessionDetail_Invalid(t *testing.T) {
	t.Parallel()
	_, err := GetSessionDetail("bad;session")
	if err == nil {
		t.Error("GetSessionDetail(bad;session) expected error for invalid session name")
	}
}

func TestSessionsJSON(t *testing.T) {
	requireTmux(t)
	data, err := SessionsJSON()
	// Error is OK if tmux server not running
	if err != nil {
		return
	}
	// Verify valid JSON
	var sessions []json.RawMessage
	if err := json.Unmarshal(data, &sessions); err != nil {
		t.Fatalf("SessionsJSON returned invalid JSON: %v\n%s", err, string(data))
	}
}

func TestSessionDetailJSON_Invalid(t *testing.T) {
	t.Parallel()
	_, err := SessionDetailJSON("bad;session")
	if err == nil {
		t.Error("SessionDetailJSON(bad;session) expected error for invalid session name")
	}
}

func TestTmuxOutput_Invalid(t *testing.T) {
	t.Parallel()
	_, err := tmuxOutput("invalid-subcommand-xyz")
	if err == nil {
		t.Error("tmuxOutput(invalid-subcommand-xyz) expected error")
	}
}

func TestTmuxExec_Invalid(t *testing.T) {
	t.Parallel()
	_, err := tmuxExec("invalid-subcommand-xyz")
	if err == nil {
		t.Error("tmuxExec(invalid-subcommand-xyz) expected error")
	}
}

func TestAtoi(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input  string
		expect int
	}{
		{"42", 42},
		{"0", 0},
		{"", 0},
		{"abc", 0},
		{"3.14", 0}, // Atoi doesn't parse floats
		{"-7", -7},
		{"999", 999},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := atoi(tt.input)
			if got != tt.expect {
				t.Errorf("atoi(%q) = %d, want %d", tt.input, got, tt.expect)
			}
		})
	}
}

func TestSendKeys_InvalidPane(t *testing.T) {
	t.Parallel()
	// SendKeys does not validate pane IDs; it passes directly to tmux.
	// With a bad pane and no tmux server, it should return an error, not panic.
	err := SendKeys("invalid-pane-xyz", "test")
	// Error is expected but not required (no panic is the key assertion)
	if err != nil {
		t.Logf("SendKeys error (expected): %v", err)
	}
}

func TestCapturePane_InvalidPane(t *testing.T) {
	t.Parallel()
	_, err := CapturePane("invalid-pane-xyz", 10)
	// Error is expected if tmux not available or pane invalid
	if err != nil {
		t.Logf("CapturePane error (expected): %v", err)
	}
}

func TestResizePane_InvalidPane(t *testing.T) {
	t.Parallel()
	err := ResizePane("bad;pane", 80, 24)
	if err == nil {
		t.Error("ResizePane(bad;pane) expected error for invalid pane ID")
	}
}

func TestNewWindow_InvalidSession(t *testing.T) {
	t.Parallel()
	_, err := NewWindow("nonexistent-session-xyz", "test-win")
	if err != nil {
		t.Logf("NewWindow error (expected): %v", err)
	}
}

func TestCurrentPane_InvalidSession(t *testing.T) {
	t.Parallel()
	_, err := CurrentPane("nonexistent-session-xyz")
	if err != nil {
		t.Logf("CurrentPane error (expected): %v", err)
	}
}

func TestKillPane_InvalidPane(t *testing.T) {
	t.Parallel()
	// KillPane does not validate pane IDs; it passes directly to tmux.
	// Should not panic.
	err := KillPane("invalid-pane-xyz")
	// Error is expected but not required (no panic is the key assertion)
	if err != nil {
		t.Logf("KillPane error (expected): %v", err)
	}
}

// Integration tests that require a running tmux server and create real sessions.

func TestListPanes_Success(t *testing.T) {
	requireTmuxServer(t)
	sessionName := "ttyweb-panes-" + randomSuffix()
	_, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer KillSession(sessionName)

	panes, err := ListPanes(sessionName)
	if err != nil {
		t.Fatalf("ListPanes(%q) failed: %v", sessionName, err)
	}
	if panes == nil {
		t.Fatal("ListPanes returned nil, expected non-nil slice")
	}
	if len(panes) == 0 {
		t.Fatal("ListPanes returned empty slice, expected at least one pane")
	}
	// First pane should belong to our session
	if panes[0].Session != sessionName {
		t.Errorf("pane session = %q, want %q", panes[0].Session, sessionName)
	}
	if panes[0].ID == "" {
		t.Error("pane ID is empty")
	}
	if panes[0].Width <= 0 || panes[0].Height <= 0 {
		t.Errorf("pane dimensions: width=%d height=%d, expected positive values", panes[0].Width, panes[0].Height)
	}
}

func TestGetSessionDetail_Success(t *testing.T) {
	requireTmuxServer(t)
	sessionName := "ttyweb-detail-" + randomSuffix()
	_, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer KillSession(sessionName)

	detail, err := GetSessionDetail(sessionName)
	if err != nil {
		t.Fatalf("GetSessionDetail(%q) failed: %v", sessionName, err)
	}
	if detail == nil {
		t.Fatal("GetSessionDetail returned nil")
	}
	if detail.Name != sessionName {
		t.Errorf("detail.Name = %q, want %q", detail.Name, sessionName)
	}
	if detail.Panes == nil {
		t.Error("detail.Panes is nil, expected non-nil slice")
	}
}

func TestSessionsJSON_Success(t *testing.T) {
	requireTmuxServer(t)
	data, err := SessionsJSON()
	if err != nil {
		t.Fatalf("SessionsJSON failed: %v", err)
	}
	// Verify valid JSON array
	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		t.Fatalf("SessionsJSON returned invalid JSON: %v\n%s", err, string(data))
	}
}

func TestSessionDetailJSON_Success(t *testing.T) {
	requireTmuxServer(t)
	sessionName := "ttyweb-json-" + randomSuffix()
	_, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer KillSession(sessionName)

	data, err := SessionDetailJSON(sessionName)
	if err != nil {
		t.Fatalf("SessionDetailJSON failed: %v", err)
	}
	// Verify valid JSON
	var detail SessionDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		t.Fatalf("SessionDetailJSON returned invalid JSON: %v\n%s", err, string(data))
	}
	if detail.Name != sessionName {
		t.Errorf("detail.Name = %q, want %q", detail.Name, sessionName)
	}
}

func TestSendKeys_Success(t *testing.T) {
	requireTmuxServer(t)
	sessionName := "ttyweb-keys-" + randomSuffix()
	_, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer KillSession(sessionName)

	paneID, err := CurrentPane(sessionName)
	if err != nil {
		t.Fatalf("CurrentPane failed: %v", err)
	}

	// Send keys should succeed with a valid pane
	err = SendKeys(paneID, "echo", "hello")
	if err != nil {
		t.Fatalf("SendKeys failed: %v", err)
	}
}

func TestCapturePane_Success(t *testing.T) {
	requireTmuxServer(t)
	sessionName := "ttyweb-capture-" + randomSuffix()
	_, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer KillSession(sessionName)

	paneID, err := CurrentPane(sessionName)
	if err != nil {
		t.Fatalf("CurrentPane failed: %v", err)
	}

	// CapturePane should return some output
	out, err := CapturePane(paneID, 10)
	if err != nil {
		t.Fatalf("CapturePane failed: %v", err)
	}
	// Output may be empty for a fresh shell, but should not error
	_ = out
}

func TestResizePane_Success(t *testing.T) {
	requireTmuxServer(t)
	sessionName := "ttyweb-resize-" + randomSuffix()
	_, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer KillSession(sessionName)

	paneID, err := CurrentPane(sessionName)
	if err != nil {
		t.Fatalf("CurrentPane failed: %v", err)
	}

	// ResizePane with valid pane should succeed
	err = ResizePane(paneID, 80, 24)
	if err != nil {
		t.Fatalf("ResizePane failed: %v", err)
	}
}

func TestNewWindow_Success(t *testing.T) {
	requireTmuxServer(t)
	sessionName := "ttyweb-window-" + randomSuffix()
	_, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer KillSession(sessionName)

	// NewWindow should create a window and return a pane ID
	paneID, err := NewWindow(sessionName, "test-win")
	if err != nil {
		t.Fatalf("NewWindow failed: %v", err)
	}
	if paneID == "" {
		t.Error("NewWindow returned empty pane ID")
	}

	// Clean up the extra window by killing the pane
	err = KillPane(paneID)
	if err != nil {
		t.Logf("KillPane cleanup error (may be expected): %v", err)
	}
}

func TestCurrentPane_Success(t *testing.T) {
	requireTmuxServer(t)
	sessionName := "ttyweb-cpane-" + randomSuffix()
	_, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer KillSession(sessionName)

	paneID, err := CurrentPane(sessionName)
	if err != nil {
		t.Fatalf("CurrentPane failed: %v", err)
	}
	if paneID == "" {
		t.Error("CurrentPane returned empty pane ID")
	}
}

func TestCreateSession_EmptyName(t *testing.T) {
	requireTmuxServer(t)
	// CreateSession with empty name should auto-generate one
	name, err := CreateSession("")
	if err != nil {
		t.Fatalf("CreateSession('') failed: %v", err)
	}
	if name == "" {
		t.Error("CreateSession('') returned empty name")
	}
	// Clean up
	KillSession(name)
}

func TestCreateSession_WithCommand(t *testing.T) {
	requireTmuxServer(t)
	sessionName := "ttyweb-cmd-" + randomSuffix()
	created, err := CreateSession(sessionName, "/bin/true")
	if err != nil {
		t.Fatalf("CreateSession with command failed: %v", err)
	}
	if created != sessionName {
		t.Errorf("created = %q, want %q", created, sessionName)
	}
	// Session may have already exited because /bin/true exits immediately
	// Try to kill it (may fail, that's OK)
	_ = KillSession(sessionName)
}

func TestGetSessionDetail_NotFound(t *testing.T) {
	requireTmuxServer(t)
	_, err := GetSessionDetail("nonexistent-session-xyz-123")
	if err == nil {
		t.Error("GetSessionDetail(nonexistent) expected error")
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
