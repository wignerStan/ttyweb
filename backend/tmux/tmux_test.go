package tmux

import (
	"syscall"
	"testing"
	"time"
)

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

func TestCloseTimeoutC_Positive(t *testing.T) {
	t.Parallel()
	s := &TmuxSlave{
		closeTimeout: 1 * time.Millisecond,
		ptyClosed:    make(chan struct{}),
	}
	ch := s.closeTimeoutC()
	// A positive timeout should return a channel that fires
	select {
	case <-ch:
		// channel fired as expected
	case <-time.After(2 * time.Second):
		t.Fatal("closeTimeoutC with positive timeout did not fire")
	}
}

func TestCloseTimeoutC_Negative(t *testing.T) {
	t.Parallel()
	s := &TmuxSlave{
		closeTimeout: -1,
		ptyClosed:    make(chan struct{}),
	}
	ch := s.closeTimeoutC()
	// A negative timeout should return a channel that never fires
	select {
	case <-ch:
		t.Fatal("closeTimeoutC with negative timeout should never fire")
	case <-time.After(50 * time.Millisecond):
		// expected: channel did not fire
	}
}

func TestCloseTimeoutC_Zero(t *testing.T) {
	t.Parallel()
	s := &TmuxSlave{
		closeTimeout: 0,
		ptyClosed:    make(chan struct{}),
	}
	ch := s.closeTimeoutC()
	// Zero timeout should fire immediately (time.After(0))
	select {
	case <-ch:
		// channel fired as expected
	case <-time.After(2 * time.Second):
		t.Fatal("closeTimeoutC with zero timeout did not fire")
	}
}

func TestNewTmuxSlave_NewSession(t *testing.T) {
	requireTmux(t)
	// Empty session = create new session. This should succeed when tmux is available.
	slave, err := NewTmuxSlave("", "", WithCloseTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("NewTmuxSlave('') failed: %v", err)
	}
	if slave == nil {
		t.Fatal("expected non-nil slave")
	}
	if slave.session != "" {
		t.Errorf("expected empty session, got %q", slave.session)
	}
	if slave.pane != "" {
		t.Errorf("expected empty pane, got %q", slave.pane)
	}

	// Verify WindowTitleVariables
	vars := slave.WindowTitleVariables()
	if vars["command"] != "tmux" {
		t.Errorf("expected command=tmux, got %v", vars["command"])
	}
	if _, ok := vars["pid"]; !ok {
		t.Error("expected pid in WindowTitleVariables")
	}

	// Clean up
	done := make(chan error, 1)
	go func() {
		done <- slave.Close()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close() timed out")
	}
}

func TestNewTmuxSlave_WithSession(t *testing.T) {
	requireTmux(t)
	requireTmuxServer(t)

	sessionName := "ttyweb-slave-test-" + randomSuffix()

	// Create a session first
	_, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// Attach to it
	slave, err := NewTmuxSlave(sessionName, "", WithCloseTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("NewTmuxSlave(%q) failed: %v", sessionName, err)
	}
	if slave == nil {
		t.Fatal("expected non-nil slave")
	}
	if slave.session != sessionName {
		t.Errorf("expected session=%q, got %q", sessionName, slave.session)
	}

	// Verify WindowTitleVariables includes session
	vars := slave.WindowTitleVariables()
	if vars["session"] != sessionName {
		t.Errorf("expected session=%q, got %v", sessionName, vars["session"])
	}

	// Clean up
	done := make(chan error, 1)
	go func() {
		done <- slave.Close()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close() timed out")
	}
}

func TestNewTmuxSlave_WithPane(t *testing.T) {
	requireTmux(t)
	requireTmuxServer(t)

	sessionName := "ttyweb-slave-pane-" + randomSuffix()

	// Create a session first
	created, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	// Get the current pane ID
	paneID, err := CurrentPane(created)
	if err != nil {
		t.Fatalf("CurrentPane failed: %v", err)
	}

	// Attach to the specific pane
	slave, err := NewTmuxSlave(sessionName, paneID, WithCloseTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("NewTmuxSlave(%q, %q) failed: %v", sessionName, paneID, err)
	}
	if slave == nil {
		t.Fatal("expected non-nil slave")
	}
	if slave.pane != paneID {
		t.Errorf("expected pane=%q, got %q", paneID, slave.pane)
	}

	// Verify WindowTitleVariables includes session and pane
	vars := slave.WindowTitleVariables()
	if vars["session"] != sessionName {
		t.Errorf("expected session=%q, got %v", sessionName, vars["session"])
	}
	if vars["pane"] != paneID {
		t.Errorf("expected pane=%q, got %v", paneID, vars["pane"])
	}

	// Clean up
	done := make(chan error, 1)
	go func() {
		done <- slave.Close()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close() timed out")
	}
}

func TestNewTmuxSlave_WithOptions(t *testing.T) {
	requireTmux(t)
	// Test that options are applied to the slave
	slave, err := NewTmuxSlave("", "",
		WithCloseSignal(syscall.SIGTERM),
		WithCloseTimeout(1*time.Second),
	)
	if err != nil {
		t.Fatalf("NewTmuxSlave with options failed: %v", err)
	}
	defer func() {
		done := make(chan error, 1)
		go func() { done <- slave.Close() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}()

	if slave.closeSignal != syscall.SIGTERM {
		t.Errorf("expected SIGTERM, got %d", slave.closeSignal)
	}
	if slave.closeTimeout != 1*time.Second {
		t.Errorf("expected 1s timeout, got %v", slave.closeTimeout)
	}
}

func TestTmuxSlave_ReadWrite(t *testing.T) {
	requireTmux(t)
	// Create a slave and test that Read/Write work (no panic)
	slave, err := NewTmuxSlave("", "", WithCloseTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("NewTmuxSlave failed: %v", err)
	}
	defer func() {
		done := make(chan error, 1)
		go func() { done <- slave.Close() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}()

	// Write should work
	buf := make([]byte, 1024)
	n, err := slave.Write([]byte("echo hello\r"))
	if err != nil {
		t.Logf("Write error (may be expected): %v", err)
	}
	if n <= 0 && err == nil {
		t.Error("Write returned 0 bytes with no error")
	}

	// Read should work (may block briefly if no output)
	readDone := make(chan struct{})
	go func() {
		_, _ = slave.Read(buf)
		close(readDone)
	}()
	select {
	case <-readDone:
		// Read returned
	case <-time.After(2 * time.Second):
		// Read is blocking, which is expected for a terminal
	}
}

func TestTmuxSlave_ResizeTerminal(t *testing.T) {
	requireTmux(t)
	slave, err := NewTmuxSlave("", "", WithCloseTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("NewTmuxSlave failed: %v", err)
	}
	defer func() {
		done := make(chan error, 1)
		go func() { done <- slave.Close() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}()

	// ResizeTerminal without pane should not panic
	err = slave.ResizeTerminal(80, 24)
	if err != nil {
		t.Logf("ResizeTerminal error (may be expected): %v", err)
	}
}

func TestTmuxSlave_ResizeTerminal_WithPane(t *testing.T) {
	requireTmux(t)
	requireTmuxServer(t)

	sessionName := "ttyweb-resize-" + randomSuffix()
	created, err := CreateSession(sessionName)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer func() { _ = KillSession(sessionName) }()

	paneID, err := CurrentPane(created)
	if err != nil {
		t.Fatalf("CurrentPane failed: %v", err)
	}

	// Create slave with pane to exercise the tmuxExec resize branch
	slave, err := NewTmuxSlave(sessionName, paneID, WithCloseTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("NewTmuxSlave failed: %v", err)
	}
	defer func() {
		done := make(chan error, 1)
		go func() { done <- slave.Close() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}()

	err = slave.ResizeTerminal(120, 40)
	if err != nil {
		t.Fatalf("ResizeTerminal with pane failed: %v", err)
	}
}

func TestTmuxSlave_Close_Timeout(t *testing.T) {
	requireTmux(t)
	// Test Close with SIGSTOP to force the timeout SIGKILL path.
	// Use a very short close timeout and SIGSTOP (which cannot be caught).
	slave, err := NewTmuxSlave("", "",
		WithCloseSignal(syscall.SIGSTOP),
		WithCloseTimeout(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewTmuxSlave failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- slave.Close()
	}()
	select {
	case err := <-done:
		// Close completed (process was killed via timeout)
		if err != nil {
			t.Logf("Close error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Close() timed out — SIGKILL path may not have fired")
	}
}
