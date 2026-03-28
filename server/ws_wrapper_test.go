package server

import (
	"testing"
)

// TestWsWrapperInterface verifies that wsWrapper implements io.Reader and io.Writer
// through its Read and Write methods. A full integration test would require
// mocking websocket.Conn, which is complex; this compilation test ensures the
// struct and method signatures remain correct.
func TestWsWrapperInterface(t *testing.T) {
	// Verify wsWrapper has Read and Write methods with correct signatures
	var _ interface {
		Read(p []byte) (n int, err error)
		Write(p []byte) (n int, err error)
	} = &wsWrapper{}
}
