package server

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// hijackableResponseWriter wraps a ResponseWriter with a Hijack method.
type hijackableResponseWriter struct {
	http.ResponseWriter
	conn net.Conn
	bw   *bufio.ReadWriter
}

func (h *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return h.conn, h.bw, nil
}

func TestLogResponseWriter_Hijack(t *testing.T) {
	serverConn, _ := net.Pipe()
	defer func() { _ = serverConn.Close() }()

	br := bufio.NewReader(serverConn)
	bw := bufio.NewWriter(serverConn)
	mockRW := bufio.NewReadWriter(br, bw)

	lrw := &logResponseWriter{
		ResponseWriter: &hijackableResponseWriter{
			ResponseWriter: httptest.NewRecorder(),
			conn:           serverConn,
			bw:             mockRW,
		},
	}

	conn, returnedBW, err := lrw.Hijack()
	if err != nil {
		t.Fatalf("Hijack failed: %v", err)
	}
	if conn != serverConn {
		t.Fatal("expected returned connection to match mock")
	}
	if returnedBW != mockRW {
		t.Fatal("expected returned bufio.ReadWriter to match mock")
	}
	if lrw.status != http.StatusSwitchingProtocols {
		t.Fatalf("expected status 101, got %d", lrw.status)
	}
}
