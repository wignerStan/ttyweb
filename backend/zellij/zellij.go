package zellij

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/pkg/errors"
)

// Slave is a zellij session attached via PTY.
// It implements server.Slave interface from ttyweb/server.
type Slave struct {
	pty          *os.File
	cmd          *exec.Cmd
	session      string
	closeSignal  syscall.Signal
	closeTimeout time.Duration
	ptyClosed    chan struct{}
	ptyMu        sync.Mutex
}

// Option configures Slave behavior.
type Option func(*Slave)

// WithCloseSignal sets the signal sent on close.
func WithCloseSignal(sig syscall.Signal) Option {
	return func(s *Slave) {
		s.closeSignal = sig
	}
}

// WithCloseTimeout sets how long to wait before SIGKILL.
func WithCloseTimeout(d time.Duration) Option {
	return func(s *Slave) {
		s.closeTimeout = d
	}
}

// NewZellijSlave creates a PTY attached to a zellij session.
// If session is empty, creates a new session.
func NewZellijSlave(session string, options ...Option) (*Slave, error) {
	args := []string{"zellij"}
	if session == "" {
		args = append(args, "new-session", "-s", "ttyweb")
	} else {
		args = append(args, "attach", session)
	}

	cmd := exec.CommandContext(context.Background(), args[0], args[1:]...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "zellij pty start failed: %v", args)
	}
	ptyClosed := make(chan struct{})

	slave := &Slave{
		session:      session,
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
			slave.ptyMu.Lock()
			_ = slave.pty.Close()
			slave.ptyMu.Unlock()
			close(slave.ptyClosed)
		}()
		_ = slave.cmd.Wait()
	}()

	return slave, nil
}

func (s *Slave) Read(p []byte) (n int, err error) {
	s.ptyMu.Lock()
	defer s.ptyMu.Unlock()
	n, err = s.pty.Read(p)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read from zellij pty")
	}
	return n, nil
}

func (s *Slave) Write(p []byte) (n int, err error) {
	s.ptyMu.Lock()
	defer s.ptyMu.Unlock()
	n, err = s.pty.Write(p)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to write to zellij pty")
	}
	return n, nil
}

// WindowTitleVariables returns template variables for the window title.
func (s *Slave) WindowTitleVariables() map[string]any {
	vars := map[string]any{
		"command": "zellij",
	}
	if s.session != "" {
		vars["session"] = s.session
	}
	vars["pid"] = s.cmd.Process.Pid
	return vars
}

// ResizeTerminal resizes the PTY to the given dimensions.
func (s *Slave) ResizeTerminal(width int, height int) error {
	s.ptyMu.Lock()
	defer s.ptyMu.Unlock()
	ws := pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
		X:    0,
		Y:    0,
	}
	err := pty.Setsize(s.pty, &ws)
	if err != nil {
		return errors.Wrapf(err, "failed to resize zellij terminal")
	}
	return nil
}

// Close terminates the zellij session attachment.
func (s *Slave) Close() error {
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

func (s *Slave) closeTimeoutC() <-chan time.Time {
	if s.closeTimeout >= 0 {
		return time.After(s.closeTimeout)
	}
	return make(chan time.Time)
}
