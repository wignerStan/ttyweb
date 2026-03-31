package tmux

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// uniqueSessionName generates a unique tmux session name for testing.
func uniqueSessionName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, rand.Intn(100000))
}

// TestFactory_GetSessionDetail_NotFound tests that GetSessionDetail returns
// an error when the session doesn't exist.
func TestFactory_GetSessionDetail_NotFound(t *testing.T) {
	f := NewFactory("")

	// Request a non-existent session — should return an error.
	_, err := f.GetSessionDetail("nonexistent-session-xyz")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

// TestFactory_GetSessionDetail_SessionExists tests GetSessionDetail with a
// real tmux session.
func TestFactory_GetSessionDetail_SessionExists(t *testing.T) {
	f := NewFactory("")

	// Create a session first.
	sessionName := uniqueSessionName("detail")
	name, err := f.CreateSession(sessionName)
	if err != nil {
		t.Skipf("skipping: cannot create tmux session: %v", err)
	}
	defer KillSession(name)

	// Get the detail.
	data, err := f.GetSessionDetail(sessionName)
	if err != nil {
		t.Fatalf("GetSessionDetail failed: %v", err)
	}
	if data == nil {
		t.Error("expected non-nil JSON data")
	}
}

// TestFactory_CreateSession_AndList tests the full create-and-list cycle.
func TestFactory_CreateSession_AndList(t *testing.T) {
	f := NewFactory("")

	sessionName := uniqueSessionName("list")
	created, err := f.CreateSession(sessionName)
	if err != nil {
		t.Skipf("skipping: cannot create tmux session: %v", err)
	}
	defer KillSession(created)

	// List should include our session.
	data, err := f.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, sessionName) {
		t.Errorf("expected session %q in list, got: %s", sessionName, body)
	}
}

// TestFactory_KillSession_NotFound tests that KillSession returns an error
// for a nonexistent session.
func TestFactory_KillSession_NotFound(t *testing.T) {
	f := NewFactory("")

	err := f.KillSession("nonexistent-session-xyz")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}
