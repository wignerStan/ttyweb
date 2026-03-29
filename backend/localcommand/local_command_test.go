package localcommand

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestNewFactory(t *testing.T) {
	factory, err := NewFactory("/bin/false", []string{}, &Options{CloseSignal: 123, CloseTimeout: 321})
	if err != nil {
		t.Errorf("NewFactory() returned error")
		return
	}
	if factory.command != "/bin/false" {
		t.Errorf("factory.command = %v, expected %v", factory.command, "/bin/false")
	}
	if !reflect.DeepEqual(factory.argv, []string{}) {
		t.Errorf("factory.argv = %v, expected %v", factory.argv, []string{})
	}
	if !reflect.DeepEqual(factory.options, &Options{CloseSignal: 123, CloseTimeout: 321}) {
		t.Errorf("factory.options = %v, expected %v", factory.options, &Options{})
	}

	slave, _ := factory.New(nil, nil)
	lcmd := slave.(*LocalCommand)
	if lcmd.closeSignal != 123 {
		t.Errorf("lcmd.closeSignal = %v, expected %v", lcmd.closeSignal, 123)
	}
	if lcmd.closeTimeout != time.Second*321 {
		t.Errorf("lcmd.closeTimeout = %v, expected %v", lcmd.closeTimeout, time.Second*321)
	}
}

func TestFactoryName(t *testing.T) {
	factory, err := NewFactory("/bin/cat", []string{}, &Options{})
	if err != nil {
		t.Fatalf("NewFactory() returned error: %v", err)
	}
	if factory.Name() != "local command" {
		t.Fatalf("expected 'local command', got %q", factory.Name())
	}
}

func TestNew_FailedCommand(t *testing.T) {
	t.Parallel()
	_, err := New("/nonexistent/command/that/does/not/exist", []string{}, nil)
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

func TestNew_WithHeaders(t *testing.T) {
	t.Parallel()
	lcmd, err := New("/bin/cat", []string{}, map[string][]string{
		"X-Custom": {"value1", "value2"},
	}, WithCloseTimeout(5*time.Second))
	if err != nil {
		t.Skipf("skipping: PTY not available: %v", err)
	}
	defer func() { _ = lcmd.Close() }()
	if lcmd.closeTimeout != 5*time.Second {
		t.Errorf("closeTimeout = %v, want 5s", lcmd.closeTimeout)
	}
}

func TestCloseTimeoutC_Negative(t *testing.T) {
	t.Parallel()
	lcmd := &LocalCommand{closeTimeout: -1 * time.Second}
	ch := lcmd.closeTimeoutC()
	select {
	case <-ch:
		t.Error("negative timeout channel should never fire")
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}

func TestFactoryNew_NegativeTimeout(t *testing.T) {
	// CloseTimeout < 0 should NOT add WithCloseTimeout option
	factory, err := NewFactory("/bin/cat", []string{}, &Options{CloseTimeout: -1})
	if err != nil {
		t.Fatalf("NewFactory() returned error: %v", err)
	}
	if len(factory.opts) != 1 {
		t.Errorf("expected 1 option (close signal only), got %d", len(factory.opts))
	}
}

func TestFactoryNew(t *testing.T) {
	factory, err := NewFactory("/bin/cat", []string{}, &Options{})
	if err != nil {
		t.Errorf("NewFactory() returned error")
		return
	}

	slave, err := factory.New(nil, nil)
	if err != nil {
		t.Errorf("factory.New() returned error")
		return
	}

	writeBuf := []byte("foobar\n")
	n, err := slave.Write(writeBuf)
	if err != nil {
		t.Errorf("write() failed: %v", err)
		return
	}
	if n != 7 {
		t.Errorf("Unexpected write length. n = %d, expected n = %d", n, 7)
		return
	}

	// Local echo is on, so we get the output twice:
	// Once because we're "typing" it, and once more
	// repeated back to us by `cat`. Also, \r\n
	// because we're a terminal.
	expectedBuf := []byte("foobar\r\nfoobar\r\n")
	readBuf := make([]byte, 1024)
	var totalRead int
	for totalRead < 16 {
		n, err = slave.Read(readBuf[totalRead:])
		if err != nil {
			t.Errorf("read() failed: %v", err)
			return
		}
		totalRead += n
	}
	if totalRead != 16 {
		t.Errorf("Unexpected read length. totalRead = %d, expected totalRead = %d", totalRead, 16)
		return
	}
	if !bytes.Equal(readBuf[:totalRead], expectedBuf) {
		t.Errorf("unexpected output from slave: got %v, expected %v", readBuf[:totalRead], expectedBuf)
	}
	err = slave.Close()
	if err != nil {
		t.Errorf("close() failed: %v", err)
		return
	}

}
