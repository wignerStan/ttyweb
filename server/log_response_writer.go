package server

import (
	"bufio"
	"net"
	"net/http"

	"github.com/pkg/errors"
)

type logResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *logResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *logResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, _ := w.ResponseWriter.(http.Hijacker)
	w.status = http.StatusSwitchingProtocols
	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to hijack connection")
	}
	return conn, rw, nil
}
