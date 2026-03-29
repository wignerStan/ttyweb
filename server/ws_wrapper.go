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
		return 0, err
	}
	defer func() { _ = writer.Close() }()
	return writer.Write(p)
}

func (wsw *wsWrapper) Read(p []byte) (n int, err error) {
	for {
		msgType, reader, err := wsw.NextReader()
		if err != nil {
			return 0, err
		}

		if msgType != websocket.TextMessage {
			continue
		}

		b, err := io.ReadAll(io.LimitReader(reader, int64(len(p))))
		if len(b) > len(p) {
			return 0, errors.Wrapf(err, "Client message exceeded buffer size")
		}
		n = copy(p, b)
		return n, err
	}
}
