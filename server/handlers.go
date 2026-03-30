package server

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"ttyweb/ai"
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

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		conn, err := server.upgrader.Upgrade(w, r, nil)
		if err != nil {
			closeReason = err.Error()
			return
		}
		defer func() { _ = conn.Close() }()

		headers := map[string][]string(nil)
		if server.options.PassHeaders {
			headers = r.Header
		}
		err = server.processWSConn(ctx, conn, headers)

		closeReason = classifyWSCloseError(err, server.factory.Name())
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

	// Build a pane key for the state machine from session/pane params.
	paneKey := params.Get("session")
	if pane := params.Get("pane"); pane != "" {
		if paneKey != "" {
			paneKey = paneKey + ":" + pane
		} else {
			paneKey = pane
		}
	}
	if paneKey == "" {
		paneKey = "default"
	}

	bridge := &aiStateMachineBridge{
		inner:   ai.NewANSITerminalInterceptor(),
		sm:      server.stateMachine,
		paneKey: paneKey,
	}
	opts = append(opts, webtty.WithOutputInterceptor(bridge))

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
	titleVars, err := server.titleVariables(
		[]string{"server", "master", "slave"},
		map[string]map[string]any{
			"server": server.options.TitleVariables,
			"master": {
				"remote_addr": conn.RemoteAddr(),
			},
			"slave": slave.WindowTitleVariables(),
		},
	)
	if err != nil {
		log.Printf("failed to build title variables: %v", err)
		return nil, errors.New("failed to build title variables")
	}

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

// classifyWSCloseError returns a human-readable close reason for a WebSocket error.
func classifyWSCloseError(err error, backendName string) string {
	switch err {
	case nil:
		return "normal close"
	case context.Canceled:
		return "cancelation"
	case webtty.ErrSlaveClosed:
		return backendName
	case webtty.ErrMasterClosed:
		return "client"
	default:
		return fmt.Sprintf("an error: %s", err)
	}
}

func (*Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

// titleVariables merges maps in a specified order.
// varUnits are name-keyed maps, whose names will be iterated using order.
func (*Server) titleVariables(order []string, varUnits map[string]map[string]any) (map[string]any, error) {
	titleVars := map[string]any{}

	for _, name := range order {
		vars, ok := varUnits[name]
		if !ok {
			return nil, fmt.Errorf("title variable name error: %s not found", name)
		}
		for key, val := range vars {
			titleVars[key] = val
		}
	}

	// safe net for conflicted keys
	for _, name := range order {
		titleVars[name] = varUnits[name]
	}

	return titleVars, nil
}

// aiStateMachineBridge connects the ANSI interceptor's pattern matching
// to the per-pane state machine's transitions.
type aiStateMachineBridge struct {
	inner   ai.OutputInterceptor
	sm      *ai.StateMachine
	paneKey string
}

// Intercept delegates to the inner interceptor and transitions the state
// machine for any ai_state_change metadata.
func (b *aiStateMachineBridge) Intercept(data []byte) []ai.Metadata {
	metadata := b.inner.Intercept(data)
	for _, m := range metadata {
		if m.Type == "ai_state_change" {
			if state, ok := m.Data["state"].(string); ok {
				b.sm.Transition(b.paneKey, ai.AIState(state))
			}
		}
	}
	return metadata
}

// registerAIStateChangeHandlers wires up the state machine to auto-create
// task events when the AI transitions to/from working state.
func registerAIStateChangeHandlers(sm *ai.StateMachine) {
	sm.OnStateChange(func(paneKey string, from, to ai.AIState) {
		if to == ai.AIStateWorking {
			slog.Info("AI started working, auto-creating task",
				"pane", paneKey,
				"from_state", string(from),
			)

			store.AddTaskEvent(&TaskEvent{
				PaneKey: paneKey,
				Event:   "ai_started_working",
				Data: map[string]any{
					"from_state": string(from),
					"pane_key":   paneKey,
				},
			})

			return
		}

		if to == ai.AIStateIdle && from == ai.AIStateWorking {
			slog.Info("AI task completed",
				"pane", paneKey,
			)

			store.AddTaskEvent(&TaskEvent{
				PaneKey: paneKey,
				Event:   "ai_completed",
				Data:    map[string]any{"pane_key": paneKey},
			})
		}
	})
}
