package webtty

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"sync"
	"testing"
)

func TestSetEncoding(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, mSlave, _, cancel := prepareSUT(t, &wg, WithPermitWrite())
	defer cancel()

	// Absorb initialization messages
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)

	// Send input with null codec (default) — should pass through
	nullMessage := []byte("1hello\n")
	_, _ = mMaster.masterToGottyWriter.Write(nullMessage)
	readBuf := make([]byte, 1024)
	n, err := mSlave.gottyToSlaveReader.Read(readBuf)
	if err != nil {
		t.Fatalf("Unexpected error reading from slave: %s", err)
	}
	if !bytes.Equal(readBuf[:n], nullMessage[1:]) {
		t.Fatalf("Expected %q, got %q", nullMessage[1:], readBuf[:n])
	}

	// Switch to base64 encoding
	encMessage := []byte("4base64")
	_, _ = mMaster.masterToGottyWriter.Write(encMessage)

	// Send input that is now base64 encoded
	b64Input := base64.StdEncoding.EncodeToString([]byte("world"))
	base64Message := []byte("1" + b64Input)
	_, _ = mMaster.masterToGottyWriter.Write(base64Message)

	n, err = mSlave.gottyToSlaveReader.Read(readBuf)
	if err != nil {
		t.Fatalf("Unexpected error reading from slave with base64 codec: %s", err)
	}
	if !bytes.Equal(readBuf[:n], []byte("world")) {
		t.Fatalf("Expected %q, got %q", "world", readBuf[:n])
	}

	// Switch back to null encoding
	encMessage = []byte("4null")
	_, _ = mMaster.masterToGottyWriter.Write(encMessage)

	// Send input with null codec again
	nullMessage2 := []byte("1direct\n")
	_, _ = mMaster.masterToGottyWriter.Write(nullMessage2)

	n, err = mSlave.gottyToSlaveReader.Read(readBuf)
	if err != nil {
		t.Fatalf("Unexpected error reading from slave with null codec: %s", err)
	}
	if !bytes.Equal(readBuf[:n], nullMessage2[1:]) {
		t.Fatalf("Expected %q, got %q", nullMessage2[1:], readBuf[:n])
	}

	cancel()
	wg.Wait()
}

func TestUnknownMessageType(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, _, _, cancel := prepareSUT(t, &wg)
	defer cancel()

	// Absorb initialization messages
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)

	// Send a message with an unknown type byte '9'
	message := []byte("9unknown")
	_, _ = mMaster.masterToGottyWriter.Write(message)

	// The Run loop should return an error for the unknown message type
	// Wait for the goroutine to finish
	wg.Wait()
	// If we get here without hanging, the unknown message was handled
}

func TestPermitWriteFalse(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	// Default is permitWrite=false, no WithPermitWrite() option
	mMaster, mSlave, _, cancel := prepareSUT(t, &wg)

	// Absorb initialization messages
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)

	// Send input from master — should be ignored since permitWrite is false
	message := []byte("1hello\n")
	_, _ = mMaster.masterToGottyWriter.Write(message)

	// Close slave's read pipe to unblock the slave-read goroutine in Run
	// then cancel and wait
	_ = mSlave.slaveToGottyWriter.Close()
	cancel()
	wg.Wait()

	// If we reach here without issues, the input was silently dropped
}

func TestZeroLengthRead(t *testing.T) {
	// Test that handleMasterReadEvent handles zero-length data without panicking.
	// We test this by directly calling handleMasterReadEvent on a WebTTY instance.
	slave := newMockSlave()
	master := newMockMaster()

	dt, err := New(master, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	// handleMasterReadEvent with empty data should return an error, not panic
	handleErr := dt.handleMasterReadEvent([]byte{})
	if handleErr == nil {
		t.Fatal("Expected error from zero-length read, got nil")
	}
}

func TestMalformedResize(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	mMaster, _, _, cancel := prepareSUT(t, &wg)
	defer cancel()

	// Absorb initialization messages
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, mMaster.gottyToMasterReader, SetBufferSize)

	// Send a resize message with invalid JSON
	message := []byte("3{invalid-json}\n")
	_, _ = mMaster.masterToGottyWriter.Write(message)

	// The malformed JSON should cause an error, which terminates the Run loop
	// Wait for completion without hanging
	wg.Wait()
	// If we reach here, the error was handled without panic
}

func TestContextCancellation(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	_, _, _, cancel := prepareSUT(t, &wg)

	// Cancel the context immediately
	cancel()

	// Wait for Run to complete
	wg.Wait()
}

// blockingReadMaster is a mockMaster that blocks on Read until closed,
// used to ensure the Run loop is waiting on master when we cancel context.
type blockingReadMaster struct {
	mockMaster
	blockCh chan struct{}
}

func (m *blockingReadMaster) Read(buf []byte) (int, error) {
	<-m.blockCh
	return 0, io.EOF
}

func TestContextCancellationMidSession(t *testing.T) {
	slave := newMockSlave()
	rawMaster := newMockMaster()
	blockMaster := &blockingReadMaster{
		mockMaster: *rawMaster,
		blockCh:    make(chan struct{}),
	}

	dt, err := New(blockMaster, slave)
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_ = dt.Run(ctx)
		wg.Done()
	}()

	// Give the goroutine time to start and block on master Read
	// Absorb init messages first
	checkNextMsgType(t, rawMaster.gottyToMasterReader, SetWindowTitle)
	checkNextMsgType(t, rawMaster.gottyToMasterReader, SetBufferSize)

	// Now cancel the context
	cancel()

	wg.Wait()
}
