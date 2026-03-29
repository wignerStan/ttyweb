package tmux

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/pkg/errors"
)

// TmuxSlave is a tmux session attached via PTY.
// It implements server.Slave interface from ttyweb/server.
type TmuxSlave struct {
	pty     *os.File
	cmd     *exec.Cmd
	session string
	pane    string

	closeSignal  syscall.Signal
	closeTimeout time.Duration

	ptyClosed chan struct{}
}

// Option configures TmuxSlave behavior.
type Option func(*TmuxSlave)

// WithCloseSignal sets the signal sent on close.
func WithCloseSignal(sig syscall.Signal) Option {
	return func(s *TmuxSlave) {
		s.closeSignal = sig
	}
}

// WithCloseTimeout sets how long to wait before SIGKILL.
func WithCloseTimeout(d time.Duration) Option {
	return func(s *TmuxSlave) {
		s.closeTimeout = d
	}
}

// NewTmuxSlave creates a PTY attached to a tmux session.
// If session is empty, creates a new session.
// If pane is empty, attaches to the current pane.
func NewTmuxSlave(session string, pane string, options ...Option) (*TmuxSlave, error) {
	args := []string{"tmux"}
	if session == "" {
		// Create new session
		args = append(args, "new-session", "-A", "-s", "ttyweb")
	} else if pane != "" {
		// Attach to specific pane
		args = append(args, "attach-session", "-t", pane)
	} else {
		// Attach to session (current window)
		args = append(args, "attach-session", "-t", session)
	}

	cmd := exec.CommandContext(context.Background(), args[0], args[1:]...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "tmux pty start failed: %v", args)
	}
	ptyClosed := make(chan struct{})

	slave := &TmuxSlave{
		session:      session,
		pane:         pane,
		closeSignal:  syscall.SIGHUP,
		closeTimeout: 10 * time.Second,
		cmd:          cmd,
		pty:          ptyFile,
		ptyClosed:    ptyClosed,
	}

	for _, opt := range options {
		opt(slave)
	}

	go func() {
		defer func() {
			_ = slave.pty.Close()
			close(slave.ptyClosed)
		}()
		_ = slave.cmd.Wait()
	}()

	return slave, nil
}

func (s *TmuxSlave) Read(p []byte) (n int, err error) {
	return s.pty.Read(p)
}

func (s *TmuxSlave) Write(p []byte) (n int, err error) {
	return s.pty.Write(p)
}

func (s *TmuxSlave) WindowTitleVariables() map[string]interface{} {
	vars := map[string]interface{}{
		"command": "tmux",
	}
	if s.session != "" {
		vars["session"] = s.session
	}
	if s.pane != "" {
		vars["pane"] = s.pane
	}
	vars["pid"] = s.cmd.Process.Pid
	return vars
}

func (s *TmuxSlave) ResizeTerminal(width int, height int) error {
	ws := pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
		X:    0,
		Y:    0,
	}
	if err := pty.Setsize(s.pty, &ws); err != nil {
		return err
	}
	// Also resize via tmux for accurate internal state
	if s.pane != "" {
		_, _ = tmuxExec("resize-pane", "-t", s.pane,
			"-x", strconv.Itoa(width), "-y", strconv.Itoa(height))
	}
	return nil
}

func (s *TmuxSlave) Close() error {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Signal(s.closeSignal)
	}
	for {
		select {
		case <-s.ptyClosed:
			return nil
		case <-s.closeTimeoutC():
			_ = s.cmd.Process.Signal(syscall.SIGKILL)
		}
	}
}

func (s *TmuxSlave) closeTimeoutC() <-chan time.Time {
	if s.closeTimeout >= 0 {
		return time.After(s.closeTimeout)
	}
	return make(chan time.Time)
}
