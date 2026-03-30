package server

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"ttyweb/pkg/validate"
	"ttyweb/webtty"
)

func (server *Server) generateHandleWS(ctx context.Context, cancel context.CancelFunc, counter *counter) http.HandlerFunc {
	once := new(int64)

	go func() {
		select {
		case <-counter.timer().C:
			cancel()
		case <-ctx.Done():
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		if server.options.Once {
			success := atomic.CompareAndSwapInt64(once, 0, 1)
			if !success {
				http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
				return
			}
		}

		num := counter.add(1)
		closeReason := "unknown reason"

		defer func() {
			num := counter.done()
			log.Printf(
				"Connection closed by %s: %s, connections: %d/%d",
				closeReason, r.RemoteAddr, num, server.options.MaxConnection,
			)

			if server.options.Once {
				cancel()
			}
		}()

		if int64(server.options.MaxConnection) != 0 {
			if num > server.options.MaxConnection {
				closeReason = "exceeding max number of connections"
				http.Error(w, "Server is busy", http.StatusServiceUnavailable)
				return
			}
		}

		log.Printf("New client connected: %s, connections: %d/%d", r.RemoteAddr, num, server.options.MaxConnection)

		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		conn, err := server.upgrader.Upgrade(w, r, nil)
		if err != nil {
			closeReason = err.Error()
			return
		}
		defer func() { _ = conn.Close() }()

		if server.options.PassHeaders {
			err = server.processWSConn(ctx, conn, r.Header)
		} else {
			err = server.processWSConn(ctx, conn, nil)
		}

		switch err {
		case ctx.Err():
			closeReason = "cancelation"
		case webtty.ErrSlaveClosed:
			closeReason = server.factory.Name()
		case webtty.ErrMasterClosed:
			closeReason = "client"
		default:
			closeReason = fmt.Sprintf("an error: %s", err)
		}
	}
}

func (server *Server) processWSConn(ctx context.Context, conn *websocket.Conn, headers map[string][]string) error {
	init, err := server.authenticateWS(conn)
	if err != nil {
		return err
	}

	params, err := server.parseInitArguments(init)
	if err != nil {
		return err
	}

	slave, err := server.factory.New(params, headers)
	if err != nil {
		return errors.Wrapf(err, "failed to create backend")
	}
	defer func() { _ = slave.Close() }()

	titleBuf, err := server.renderTitle(conn, slave)
	if err != nil {
		return err
	}

	opts := server.webttyOptions(titleBuf.Bytes())
	tty, err := webtty.New(&wsWrapper{conn}, slave, opts...)
	if err != nil {
		return errors.Wrapf(err, "failed to create webtty")
	}

	err = tty.Run(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to run webtty")
	}
	return nil
}

// authenticateWS reads and validates the initial WebSocket message.
func (server *Server) authenticateWS(conn *websocket.Conn) (*InitMessage, error) {
	typ, initLine, err := conn.ReadMessage()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to authenticate websocket connection")
	}
	if typ != websocket.TextMessage {
		return nil, errors.New("failed to authenticate websocket connection: invalid message type")
	}

	var init InitMessage
	if err := json.Unmarshal(initLine, &init); err != nil {
		return nil, errors.Wrapf(err, "failed to authenticate websocket connection")
	}
	if subtle.ConstantTimeCompare([]byte(init.AuthToken), []byte(server.options.Credential)) != 1 {
		return nil, errors.New("failed to authenticate websocket connection")
	}

	return &init, nil
}

// parseInitArguments parses and validates the init message's query arguments.
func (server *Server) parseInitArguments(init *InitMessage) (url.Values, error) {
	queryPath := "?"
	if server.options.PermitArguments && init.Arguments != "" {
		queryPath = init.Arguments
	}

	query, err := url.Parse(queryPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse arguments")
	}
	params := query.Query()

	if session := params.Get("session"); session != "" {
		if err := validate.SessionName(session); err != nil {
			return nil, errors.Wrapf(err, "invalid session parameter")
		}
	}
	if pane := params.Get("pane"); pane != "" {
		if err := validate.PaneID(pane); err != nil {
			return nil, errors.Wrapf(err, "invalid pane parameter")
		}
	}

	return params, nil
}

// renderTitle executes the title template with server, master, and slave variables.
func (server *Server) renderTitle(conn *websocket.Conn, slave Slave) (*bytes.Buffer, error) {
	titleVars := server.titleVariables(
		[]string{"server", "master", "slave"},
		map[string]map[string]any{
			"server": server.options.TitleVariables,
			"master": {
				"remote_addr": conn.RemoteAddr(),
			},
			"slave": slave.WindowTitleVariables(),
		},
	)

	titleBuf := new(bytes.Buffer)
	if err := server.titleTemplate.Execute(titleBuf, titleVars); err != nil {
		return nil, errors.Wrapf(err, "failed to fill window title template")
	}
	return titleBuf, nil
}

// webttyOptions builds the webtty.Option slice from server configuration.
func (server *Server) webttyOptions(title []byte) []webtty.Option {
	opts := []webtty.Option{webtty.WithWindowTitle(title)}
	if server.options.PermitWrite {
		opts = append(opts, webtty.WithPermitWrite())
	}
	if server.options.EnableReconnect {
		opts = append(opts, webtty.WithReconnect(server.options.ReconnectTime))
	}
	if server.options.Width > 0 {
		opts = append(opts, webtty.WithFixedColumns(server.options.Width))
	}
	if server.options.Height > 0 {
		opts = append(opts, webtty.WithFixedRows(server.options.Height))
	}
	return opts
}

func (*Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

// titleVariables merges maps in a specified order.
// varUnits are name-keyed maps, whose names will be iterated using order.
func (*Server) titleVariables(order []string, varUnits map[string]map[string]any) map[string]any {
	titleVars := map[string]any{}

	for _, name := range order {
		vars, ok := varUnits[name]
		if !ok {
			panic("title variable name error")
		}
		for key, val := range vars {
			titleVars[key] = val
		}
	}

	// safe net for conflicted keys
	for _, name := range order {
		titleVars[name] = varUnits[name]
	}

	return titleVars
}
