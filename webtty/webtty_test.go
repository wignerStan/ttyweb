package webtty

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"sync"
	"testing"
)

func TestInitialization(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, _, _, cancel := prepareSUT(t, &wg)
	defer cancel()

	// Check that the initialization happens as expected
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)
}

func TestInitializationWithPreferences(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, _, _, cancel := prepareSUT(t, &wg, WithMasterPreferences(map[string]string{"foo": "bar"}))
	defer cancel()

	// Check that the initialization happens as expected
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetPreferences)
}

func TestInitializationWithReconnect(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, _, _, cancel := prepareSUT(t, &wg, WithReconnect(10))
	defer cancel()

	// Check that the initialization happens as expected
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetReconnect)
}

func TestWriteFromSlaveCommand(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, mSlave, _, cancel := prepareSUT(t, &wg)
	defer cancel()

	// Check that the initialization happens as expected
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)

	// Simulate the slave (the process being run by GoTTY)
	// echoing "foobar"
	message := []byte("foobar")
	_, _ = mSlave.slaveToGottyWriter.Write(message)

	// And then make sure it makes it way to the client
	// through the websocket as an output message
	buf := make([]byte, 1024)
	n, err := mMaster.gottyToMasterReader.Read(buf)
	if err != nil {
		t.Fatalf("Unexpected error from Read(): %s", err)
	}
	if buf[0] != Output {
		t.Fatalf("Unexpected message type `%c`", buf[0])
	}

	// Decode it and make sure it's intact
	decoded := make([]byte, 1024)
	n, err = base64.StdEncoding.Decode(decoded, buf[1:n])
	if err != nil {
		t.Fatalf("Unexpected error from Decode(): %s", err)
	}
	if !bytes.Equal(decoded[:n], message) {
		t.Fatalf("Unexpected message received: `%s`", decoded[:n])
	}

	cancel()
	wg.Wait()
}
func TestWriteFromFrontend(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, mSlave, _, cancel := prepareSUT(t, &wg, WithPermitWrite())
	defer cancel()

	// Absorb initialization messages
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)

	// simulate input from frontend...
	message := []byte("1hello\n") // line buffered canonical mode
	_, _ = mMaster.masterToGottyWriter.Write(message)

	// ...and make sure it makes it through to the slave intact
	readBuf := make([]byte, 1024)
	n, err := mSlave.gottyToSlaveReader.Read(readBuf)
	if err != nil {
		t.Fatalf("Unexpected error from Write(): %s", err)
	}
	if !bytes.Equal(readBuf[:n], message[1:]) {
		t.Fatalf("Unexpected message received: `%s`", readBuf[:n])
	}
}

func TestPing(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, _, _, cancel := prepareSUT(t, &wg)
	defer cancel()

	// Absorb initialization messages
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)

	// ping
	message := []byte("2\n") // line buffered canonical mode
	n, err := mMaster.masterToGottyWriter.Write(message)
	if err != nil {
		t.Fatalf("Unexpected error from Write(): %s", err)
	}
	if n != len(message) {
		t.Fatalf("Write() accepted `%d` for message `%s`", n, message)
	}

	readBuf := make([]byte, 1024)
	n, err = mMaster.gottyToMasterReader.Read(readBuf)
	if err != nil {
		t.Fatalf("Unexpected error from Read(): %s", err)
	}
	if !bytes.Equal(readBuf[:n], []byte{'2'}) {
		t.Fatalf("Unexpected message received: `%s`", readBuf[:n])
	}

	cancel()
	wg.Wait()
}

func TestResizeTerminal(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, mSlave, _, cancel := prepareSUT(t, &wg)
	defer cancel()

	// Absorb initialization messages
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)

	message := []byte(`3{"Columns": 1234, "Rows": 2345}` + "\n") // line buffered canonical mode

	mSlave.wg.Add(1)
	n, err := mMaster.masterToGottyWriter.Write(message)
	if err != nil {
		t.Fatalf("Unexpected error from Write(): %s", err)
	}
	if n != len(message) {
		t.Fatalf("Write() accepted `%d` for message `%s`", n, message)
	}
	mSlave.wg.Wait()

	if mSlave.columns != 1234 {
		t.Fatalf("Columns not set correctly. Expected %v, got %v", 1234, mSlave.columns)
	}

	if mSlave.rows != 2345 {
		t.Fatalf("Rows not set correctly. Expected %v, got %v", 2345, mSlave.columns)
	}

	cancel()
	wg.Wait()
}

type mockMaster struct {
	gottyToMasterReader *io.PipeReader
	gottyToMasterWriter *io.PipeWriter
	masterToGottyReader *io.PipeReader
	masterToGottyWriter *io.PipeWriter
}

type mockSlave struct {
	gottyToSlaveReader *io.PipeReader
	gottyToSlaveWriter *io.PipeWriter
	slaveToGottyReader *io.PipeReader
	slaveToGottyWriter *io.PipeWriter
	wg                 sync.WaitGroup
	columns, rows      int
}

func prepareSUT(t *testing.T, wg *sync.WaitGroup, options ...Option) (*mockMaster, *mockSlave, *WebTTY, context.CancelFunc) {
	mMaster := newMockMaster()
	mSlave := newMockSlave()

	dt, err := New(mMaster, mSlave, options...)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		wg.Done()
		_ = dt.Run(ctx)
	}()
	return mMaster, mSlave, dt, cancel
}

func checkNextMsgType(t *testing.T, reader io.Reader, expected byte) {
	msgType, _ := nextMsg(t, reader)
	if msgType != expected {
		t.Fatalf("Unexpected message type `%c`", msgType)
	}
}

func nextMsg(t *testing.T, reader io.Reader) (byte, []byte) {
	buf := make([]byte, 1024)
	_, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	return buf[0], buf[1:]
}

func newMockMaster() *mockMaster {
	rv := &mockMaster{}
	rv.gottyToMasterReader, rv.gottyToMasterWriter = io.Pipe()
	rv.masterToGottyReader, rv.masterToGottyWriter = io.Pipe()
	return rv
}

func (mm *mockMaster) Read(buf []byte) (int, error) {
	return mm.masterToGottyReader.Read(buf)
}

func (mm *mockMaster) Write(buf []byte) (int, error) {
	return mm.gottyToMasterWriter.Write(buf)
}

func newMockSlave() *mockSlave {
	rv := &mockSlave{}
	rv.gottyToSlaveReader, rv.gottyToSlaveWriter = io.Pipe()
	rv.slaveToGottyReader, rv.slaveToGottyWriter = io.Pipe()
	return rv
}

func (ms *mockSlave) Read(buf []byte) (int, error) {
	return ms.slaveToGottyReader.Read(buf)
}

func (ms *mockSlave) Write(buf []byte) (int, error) {
	return ms.gottyToSlaveWriter.Write(buf)
}

func (ms *mockSlave) WindowTitleVariables() map[string]interface{} {
	return nil
}

func (ms *mockSlave) ResizeTerminal(columns int, rows int) error {
	ms.columns = columns
	ms.rows = rows
	ms.wg.Done()
	return nil
}

// --- Error-returning and specialized mocks ---

// errMaster is a mockMaster whose Write always returns an error.
type errMaster struct {
	mockMaster
	writeErr error
}

func (m *errMaster) Write(buf []byte) (int, error) {
	return 0, m.writeErr
}

// errSlave is a mockSlave whose Write always returns an error.
type errSlave struct {
	mockSlave
	writeErr error
}

func (s *errSlave) Write(buf []byte) (int, error) {
	return 0, s.writeErr
}

// errResizeSlave is a mockSlave whose ResizeTerminal returns an error.
type errResizeSlave struct {
	mockSlave
	resizeErr error
}

func (s *errResizeSlave) ResizeTerminal(columns int, rows int) error {
	s.columns = columns
	s.rows = rows
	return s.resizeErr
}

// eofReadMaster is a mockMaster whose Read returns EOF immediately,
// causing the master-read goroutine to exit. Write succeeds so
// sendInitializeMessage doesn't block.
type eofReadMaster struct {
	writeTo *io.PipeWriter
}

func newEOFReadMaster() *eofReadMaster {
	r, w := io.Pipe()
	go func() {
		// Drain all writes so they never block
		buf := make([]byte, 4096)
		for {
			if _, err := r.Read(buf); err != nil {
				return
			}
		}
	}()
	return &eofReadMaster{writeTo: w}
}

func (m *eofReadMaster) Read(buf []byte) (int, error) {
	return 0, io.EOF
}

func (m *eofReadMaster) Write(buf []byte) (int, error) {
	return m.writeTo.Write(buf)
}

// eofReadSlave is a mockSlave whose Read returns EOF immediately,
// causing the slave-read goroutine to exit. Write succeeds so
// handleMasterReadEvent Input writes don't block.
type eofReadSlave struct {
	writeTo *io.PipeWriter
}

func newEOFReadSlave() *eofReadSlave {
	r, w := io.Pipe()
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := r.Read(buf); err != nil {
				return
			}
		}
	}()
	return &eofReadSlave{writeTo: w}
}

func (s *eofReadSlave) Read(buf []byte) (int, error) {
	return 0, io.EOF
}

func (s *eofReadSlave) Write(buf []byte) (int, error) {
	return s.writeTo.Write(buf)
}

func (s *eofReadSlave) WindowTitleVariables() map[string]interface{} {
	return nil
}

func (s *eofReadSlave) ResizeTerminal(columns int, rows int) error {
	return nil
}

// --- Edge case tests ---

func TestNewOptionError(t *testing.T) {
	t.Parallel()

	errOption := func(wt *WebTTY) error {
		return errors.New("option error")
	}

	_, err := New(newMockMaster(), newMockSlave(), errOption)
	if err == nil {
		t.Fatal("expected error from New() when option returns error")
	}
}

func TestRunInitializationError(t *testing.T) {
	t.Parallel()

	// Master whose Write fails, causing sendInitializeMessage to fail
	master := &errMaster{
		writeErr: errors.New("write failed"),
	}
	slave := newMockSlave()

	dt, err := New(master, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = dt.Run(ctx)
	if err == nil {
		t.Fatal("expected error from Run() when init write fails")
	}
}

func TestRunSlaveReadError(t *testing.T) {
	t.Parallel()

	// Use a regular mockMaster (blocks on Read) and an eofReadSlave (returns EOF).
	// This ensures the slave-read goroutine exits first with ErrSlaveClosed.
	// We need to drain the master write pipe so sendInitializeMessage doesn't block.
	rawMaster := newMockMaster()
	slave := newEOFReadSlave()

	dt, err := New(rawMaster, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Drain init messages in background so sendInitializeMessage completes
	go func() {
		buf := make([]byte, 1024)
		rawMaster.gottyToMasterReader.Read(buf)
		rawMaster.gottyToMasterReader.Read(buf)
	}()

	err = dt.Run(ctx)
	if err == nil {
		t.Fatal("expected error from Run() when slave read fails")
	}
	if err != ErrSlaveClosed {
		t.Fatalf("expected ErrSlaveClosed, got %v", err)
	}
}

func TestRunMasterReadError(t *testing.T) {
	t.Parallel()

	// Use eofReadMaster (returns EOF on Read) so the master-read
	// goroutine exits immediately with ErrMasterClosed.
	// Use regular mockSlave (blocks on Read) so the slave-read
	// goroutine stays alive, making the result deterministic.
	master := newEOFReadMaster()
	slave := newMockSlave()

	dt, err := New(master, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = dt.Run(ctx)
	if err == nil {
		t.Fatal("expected error from Run() when master read fails")
	}
	if err != ErrMasterClosed {
		t.Fatalf("expected ErrMasterClosed, got %v", err)
	}
}

func TestRunErrorFromErrsChannel(t *testing.T) {
	t.Parallel()

	// When the errs channel returns an error (not context cancel),
	// the select case err = <-errs should be taken
	master := newEOFReadMaster()
	slave := newEOFReadSlave()

	dt, err := New(master, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = dt.Run(ctx)
	if err == nil {
		t.Fatal("expected error from Run() via errs channel")
	}
}

func TestHandleSlaveReadEventWriteError(t *testing.T) {
	t.Parallel()

	// Create a WebTTY directly and test handleSlaveReadEvent
	master := &errMaster{
		writeErr: errors.New("master write failed"),
	}
	slave := newMockSlave()

	dt, err := New(master, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	err = dt.handleSlaveReadEvent([]byte("test data"))
	if err == nil {
		t.Fatal("expected error from handleSlaveReadEvent when master write fails")
	}
}

func TestInputWithEmptyData(t *testing.T) {
	t.Parallel()

	// Test that Input with only the prefix byte (no data) is handled gracefully
	slave := newMockSlave()
	master := newMockMaster()

	dt, err := New(master, slave, WithPermitWrite())
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	// handleMasterReadEvent with just the Input prefix byte
	err = dt.handleMasterReadEvent([]byte{Input})
	if err != nil {
		t.Fatalf("Unexpected error from handleMasterReadEvent with empty input: %s", err)
	}
}

func TestInputWithSlaveWriteError(t *testing.T) {
	t.Parallel()

	master := newMockMaster()
	slave := &errSlave{
		writeErr: errors.New("slave write failed"),
	}

	dt, err := New(master, slave, WithPermitWrite())
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	err = dt.handleMasterReadEvent([]byte("1hello"))
	if err == nil {
		t.Fatal("expected error from handleMasterReadEvent when slave write fails")
	}
}

func TestPingWriteError(t *testing.T) {
	t.Parallel()

	master := &errMaster{
		writeErr: errors.New("master write failed"),
	}
	slave := newMockSlave()

	dt, err := New(master, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	err = dt.handleMasterReadEvent([]byte{Ping})
	if err == nil {
		t.Fatal("expected error from handleMasterReadEvent when pong write fails")
	}
}

func TestHandleMasterReadEvent_DecodeError(t *testing.T) {
	t.Parallel()

	master := newMockMaster()
	slave := newMockSlave()

	dt, err := New(master, slave, WithPermitWrite())
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	// Switch to base64 encoding, then send Input with invalid base64 data.
	err = dt.handleMasterReadEvent(append([]byte{SetEncoding}, []byte("base64")...))
	if err != nil {
		t.Fatalf("unexpected error setting encoding: %v", err)
	}

	// Send Input prefix byte followed by invalid base64 data.
	// base64.StdEncoding should reject '!' characters.
	err = dt.handleMasterReadEvent(append([]byte{Input}, []byte("!!!invalid!!!")...))
	if err == nil {
		t.Fatal("expected error from handleMasterReadEvent for invalid base64 payload")
	}
}

func TestResizeTerminalWithFixedColumnsAndRows(t *testing.T) {
	t.Parallel()

	master := newMockMaster()
	slave := newMockSlave()

	dt, err := New(master, slave, WithFixedColumns(80), WithFixedRows(24))
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	// With fixed columns and rows, ResizeTerminal should be a no-op
	err = dt.handleMasterReadEvent([]byte(`3{"Columns": 999, "Rows": 888}`))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	// Slave should not have been resized
	if slave.columns != 0 || slave.rows != 0 {
		t.Fatalf("expected no resize, got columns=%d rows=%d", slave.columns, slave.rows)
	}
}

func TestResizeTerminalEmptyPayload(t *testing.T) {
	t.Parallel()

	slave := newMockSlave()
	master := newMockMaster()

	dt, err := New(master, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	// ResizeTerminal with only the prefix byte and no payload
	err = dt.handleMasterReadEvent([]byte{ResizeTerminal})
	if err == nil {
		t.Fatal("expected error from handleMasterReadEvent for empty resize payload")
	}
}

func TestResizeTerminalSlaveError(t *testing.T) {
	t.Parallel()

	master := newMockMaster()
	slave := &errResizeSlave{
		resizeErr: errors.New("resize failed"),
	}

	dt, err := New(master, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	err = dt.handleMasterReadEvent([]byte(`3{"Columns": 100, "Rows": 50}`))
	if err == nil {
		t.Fatal("expected error from handleMasterReadEvent when slave resize fails")
	}
}

func TestInitializationWithReconnectWriteError(t *testing.T) {
	t.Parallel()

	// Master that succeeds for first 2 writes (WindowTitle, SetBufferSize)
	// but fails on the 3rd (SetReconnect)
	rawMaster := newMockMaster()
	master := &countingErrMaster{
		mockMaster: *rawMaster,
		failAfter:  2,
		writeErr:   errors.New("write failed on reconnect"),
	}

	// Drain the pipe so the first 2 writes don't block
	go func() {
		buf := make([]byte, 1024)
		rawMaster.gottyToMasterReader.Read(buf)
		rawMaster.gottyToMasterReader.Read(buf)
	}()

	slave := newMockSlave()

	dt, err := New(master, slave, WithReconnect(10))
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = dt.Run(ctx)
	if err == nil {
		t.Fatal("expected error from Run() when reconnect write fails")
	}
}

func TestInitializationWithPreferencesWriteError(t *testing.T) {
	t.Parallel()

	// Master that succeeds for first 2 writes but fails on the 3rd (SetPreferences)
	rawMaster := newMockMaster()
	master := &countingErrMaster{
		mockMaster: *rawMaster,
		failAfter:  2,
		writeErr:   errors.New("write failed on preferences"),
	}

	// Drain the pipe so the first 2 writes don't block
	go func() {
		buf := make([]byte, 1024)
		rawMaster.gottyToMasterReader.Read(buf)
		rawMaster.gottyToMasterReader.Read(buf)
	}()

	slave := newMockSlave()

	dt, err := New(master, slave, WithMasterPreferences(map[string]string{"a": "b"}))
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = dt.Run(ctx)
	if err == nil {
		t.Fatal("expected error from Run() when preferences write fails")
	}
}

func TestInitializationBufferSizeWriteError(t *testing.T) {
	t.Parallel()

	// Master that succeeds for the first write (WindowTitle)
	// but fails on the 2nd (SetBufferSize)
	rawMaster := newMockMaster()
	master := &countingErrMaster{
		mockMaster: *rawMaster,
		failAfter:  1,
		writeErr:   errors.New("write failed on buffer size"),
	}

	// Drain the pipe so the first write doesn't block
	go func() {
		buf := make([]byte, 1024)
		rawMaster.gottyToMasterReader.Read(buf)
	}()

	slave := newMockSlave()

	dt, err := New(master, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = dt.Run(ctx)
	if err == nil {
		t.Fatal("expected error from Run() when buffer size write fails")
	}
}

// countingErrMaster is a master that succeeds for the first N writes then fails.
type countingErrMaster struct {
	mockMaster
	count     int
	failAfter int
	writeErr  error
}

func (m *countingErrMaster) Write(buf []byte) (int, error) {
	m.count++
	if m.count > m.failAfter {
		return 0, m.writeErr
	}
	return m.mockMaster.Write(buf)
}
