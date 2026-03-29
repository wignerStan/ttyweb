package server

import (
	"io"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type wsWrapper struct {
	*websocket.Conn
}

func (wsw *wsWrapper) Write(p []byte) (n int, err error) {
	writer, err := wsw.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get websocket writer")
	}
	defer func() { _ = writer.Close() }()
	n, err = writer.Write(p)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to write to websocket")
	}
	return n, nil
}

func (wsw *wsWrapper) Read(p []byte) (n int, err error) {
	for {
		msgType, reader, err := wsw.NextReader()
		if err != nil {
			return 0, errors.Wrapf(err, "failed to get websocket reader")
		}

		if msgType != websocket.TextMessage {
			continue
		}

		b, err := io.ReadAll(io.LimitReader(reader, int64(len(p))))
		if err != nil {
			return 0, errors.Wrapf(err, "failed to read websocket message")
		}
		if len(b) > len(p) {
			return 0, errors.Wrapf(err, "Client message exceeded buffer size")
		}
		n = copy(p, b)
		return n, nil
	}
}
