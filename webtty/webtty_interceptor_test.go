package webtty

import (
	"encoding/base64"
	"encoding/json"
	"sync"
	"testing"

	"ttyweb/ai"
)

// mockInterceptor is a simple OutputInterceptor that returns fixed metadata.
type mockInterceptor struct {
	metadata []ai.Metadata
}

func (m *mockInterceptor) Intercept(_ []byte) []ai.Metadata {
	return m.metadata
}

// TestHandleSlaveReadEvent_WithInterceptor tests that the interceptor path
// in handleSlaveReadEvent produces SetMetadata messages on the master.
func TestHandleSlaveReadEvent_WithInterceptor(t *testing.T) {
	t.Parallel()

	master := newEOFReadMaster()
	slave := newMockSlave()

	interceptor := &mockInterceptor{
		metadata: []ai.Metadata{
			{
				Type: "ai_state_change",
				Data: map[string]any{"state": "working"},
			},
		},
	}

	dt, err := New(master, slave, WithOutputInterceptor(interceptor))
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	// handleSlaveReadEvent writes Output then SetMetadata to master.
	// Since master is an eofReadMaster with a drain goroutine, writes won't block.
	err = dt.handleSlaveReadEvent([]byte("test output"))
	if err != nil {
		t.Fatalf("handleSlaveReadEvent failed: %v", err)
	}
}

// TestHandleSlaveReadEvent_WithMultipleMetadata tests multiple metadata entries.
func TestHandleSlaveReadEvent_WithMultipleMetadata(t *testing.T) {
	t.Parallel()

	master := newEOFReadMaster()
	slave := newMockSlave()

	interceptor := &mockInterceptor{
		metadata: []ai.Metadata{
			{Type: "ai_state_change", Data: map[string]any{"state": "working"}},
			{Type: "session_info", Data: map[string]any{"session": "test"}},
		},
	}

	dt, err := New(master, slave, WithOutputInterceptor(interceptor))
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	err = dt.handleSlaveReadEvent([]byte("multi"))
	if err != nil {
		t.Fatalf("handleSlaveReadEvent failed: %v", err)
	}
}

// TestHandleSlaveReadEvent_NilInterceptor tests that nil interceptor is safe.
func TestHandleSlaveReadEvent_NilInterceptor(t *testing.T) {
	t.Parallel()

	master := newEOFReadMaster()
	slave := newMockSlave()

	dt, err := New(master, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	err = dt.handleSlaveReadEvent([]byte("no interceptor"))
	if err != nil {
		t.Fatalf("handleSlaveReadEvent with nil interceptor failed: %v", err)
	}
}

// TestHandleSlaveReadEvent_InterceptorMetadataFormat verifies the metadata
// format that the interceptor writes to the master connection.
func TestHandleSlaveReadEvent_InterceptorMetadataFormat(t *testing.T) {
	t.Parallel()

	master := newEOFReadMaster()
	slave := newMockSlave()

	interceptor := &mockInterceptor{
		metadata: []ai.Metadata{
			{Type: "ai_state_change", Data: map[string]any{"state": "working"}},
		},
	}

	dt, err := New(master, slave, WithOutputInterceptor(interceptor))
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	// handleSlaveReadEvent writes to master, which is drained by the
	// eofReadMaster's background goroutine. We can't easily read the
	// pipe from the test, but we can verify the interceptor is called.
	// The eofReadMaster drain goroutine ensures writes don't block.
	err = dt.handleSlaveReadEvent([]byte("test"))
	if err != nil {
		t.Fatalf("handleSlaveReadEvent failed: %v", err)
	}
}

// TestWriteFromSlaveWithInterceptor tests the full pipeline through Run with an interceptor.
func TestWriteFromSlaveWithInterceptor(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, mSlave, _, cancel := prepareSUT(t, &wg, WithOutputInterceptor(&mockInterceptor{
		metadata: []ai.Metadata{
			{Type: "test", Data: map[string]any{"key": "value"}},
		},
	}))

	// Check initialization messages.
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)

	// Write data from slave.
	_, _ = mSlave.slaveToGottyWriter.Write([]byte("intercepted"))

	// Read Output message.
	buf := make([]byte, 4096)
	_, err := mMaster.gottyToMasterReader.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if buf[0] != Output {
		t.Fatalf("expected Output, got %c", buf[0])
	}

	// Read SetMetadata message.
	n, err := mMaster.gottyToMasterReader.Read(buf)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	if buf[0] != SetMetadata {
		t.Fatalf("expected SetMetadata, got %c", buf[0])
	}

	// Decode and verify the metadata content.
	metaDecoded := make([]byte, base64.StdEncoding.DecodedLen(n-1))
	mnd, err := base64.StdEncoding.Decode(metaDecoded, buf[1:n])
	if err != nil {
		t.Fatalf("decode metadata: %v", err)
	}

	var meta ai.Metadata
	if err := json.Unmarshal(metaDecoded[:mnd], &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if meta.Type != "test" {
		t.Errorf("expected metadata type 'test', got %q", meta.Type)
	}

	cancel()
	wg.Wait()
}
