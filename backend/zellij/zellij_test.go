package zellij

import (
	"testing"
	"time"
)

func TestWithCloseSignal(t *testing.T) {
	t.Parallel()
	var slave ZellijSlave
	opt := WithCloseSignal(9)
	opt(&slave)
	if slave.closeSignal != 9 {
		t.Errorf("closeSignal = %v, want 9", slave.closeSignal)
	}
}

func TestWithCloseTimeout(t *testing.T) {
	t.Parallel()
	var slave ZellijSlave
	opt := WithCloseTimeout(5 * time.Second)
	opt(&slave)
	if slave.closeTimeout != 5*time.Second {
		t.Errorf("closeTimeout = %v, want 5s", slave.closeTimeout)
	}
}

func TestCloseTimeoutC_Positive(t *testing.T) {
	t.Parallel()
	slave := &ZellijSlave{closeTimeout: 1 * time.Millisecond}
	ch := slave.closeTimeoutC()
	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
		t.Error("closeTimeoutC should have fired")
	}
}

func TestCloseTimeoutC_Negative(t *testing.T) {
	t.Parallel()
	slave := &ZellijSlave{closeTimeout: -1 * time.Second}
	ch := slave.closeTimeoutC()
	select {
	case <-ch:
		t.Error("negative timeout channel should never fire")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestNewZellijSlave_NoZellij(t *testing.T) {
	t.Parallel()
	_, err := NewZellijSlave("test-session")
	if err == nil {
		t.Log("NewZellijSlave succeeded (zellij available)")
	} else {
		t.Logf("NewZellijSlave failed as expected: %v", err)
	}
}

func TestNewZellijSlave_EmptySession(t *testing.T) {
	t.Parallel()
	_, err := NewZellijSlave("")
	if err == nil {
		t.Log("NewZellijSlave with empty session succeeded")
	} else {
		t.Logf("NewZellijSlave with empty session failed: %v", err)
	}
}

func TestNewZellijSlave_WithOptions(t *testing.T) {
	t.Parallel()
	_, err := NewZellijSlave("test",
		WithCloseSignal(9),
		WithCloseTimeout(2*time.Second),
	)
	if err == nil {
		t.Log("NewZellijSlave with options succeeded")
	} else {
		t.Logf("NewZellijSlave with options failed: %v", err)
	}
}

func TestSlave_WindowTitleVariables(t *testing.T) {
	requireZellij(t)

	slave, err := NewZellijSlave("ttyweb-ttv-" + randomHex(4))
	if err != nil {
		t.Skipf("skipping: could not create slave: %v", err)
	}
	defer slave.Close()

	vars := slave.WindowTitleVariables()
	if vars["command"] != "zellij" {
		t.Errorf("WindowTitleVariables command = %v, want zellij", vars["command"])
	}
	if vars["session"] != slave.session {
		t.Errorf("WindowTitleVariables session = %v, want %q", vars["session"], slave.session)
	}
	if vars["pid"] == nil {
		t.Error("WindowTitleVariables pid is nil")
	}
}

func TestSlave_WindowTitleVariables_EmptySession(t *testing.T) {
	requireZellij(t)

	slave, err := NewZellijSlave("")
	if err != nil {
		t.Skipf("skipping: could not create slave: %v", err)
	}
	defer slave.Close()

	vars := slave.WindowTitleVariables()
	if vars["command"] != "zellij" {
		t.Errorf("WindowTitleVariables command = %v, want zellij", vars["command"])
	}
	if _, ok := vars["session"]; ok {
		t.Error("WindowTitleVariables should not have session key for empty session")
	}
}

func TestSlave_ResizeTerminal(t *testing.T) {
	requireZellij(t)

	slave, err := NewZellijSlave("ttyweb-rsz-" + randomHex(4))
	if err != nil {
		t.Skipf("skipping: could not create slave: %v", err)
	}
	defer slave.Close()

	err = slave.ResizeTerminal(80, 24)
	if err != nil {
		t.Errorf("ResizeTerminal returned error: %v", err)
	}
}

func TestSlave_ReadWrite(t *testing.T) {
	requireZellij(t)

	slave, err := NewZellijSlave("ttyweb-rw-" + randomHex(4))
	if err != nil {
		t.Skipf("skipping: could not create slave: %v", err)
	}
	defer slave.Close()

	// Write some data to the PTY
	n, err := slave.Write([]byte("hello"))
	if err != nil {
		t.Logf("Write returned error (may be expected): %v", err)
	} else if n != 5 {
		t.Errorf("Write returned %d bytes, want 5", n)
	}

	// Read with a short timeout to avoid blocking forever
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		n, err := slave.Read(buf)
		if err == nil {
			t.Logf("Read %d bytes from PTY", n)
		}
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Log("Read timed out (no data from terminal, expected)")
	}
}

func TestSlave_Close(t *testing.T) {
	requireZellij(t)

	slave, err := NewZellijSlave("ttyweb-cls-" + randomHex(4),
		WithCloseTimeout(1*time.Second),
	)
	if err != nil {
		t.Skipf("skipping: could not create slave: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := slave.Close()
		if err != nil {
			t.Errorf("Close returned error: %v", err)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("Close timed out after 5s")
	}
}
