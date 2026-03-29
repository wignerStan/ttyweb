package localcommand

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/pkg/errors"
)

const (
	DefaultCloseSignal  = syscall.SIGINT
	DefaultCloseTimeout = 10 * time.Second
)

type LocalCommand struct {
	command string
	argv    []string

	closeSignal  syscall.Signal
	closeTimeout time.Duration

	cmd       *exec.Cmd
	pty       *os.File
	ptyClosed chan struct{}
	ptyMu     sync.Mutex
}

func New(command string, argv []string, headers map[string][]string, options ...Option) (*LocalCommand, error) {
	cmd := exec.CommandContext(context.Background(), command, argv...)

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Combine headers into key=value pairs to set as env vars
	// Prefix the headers with "http_" so we don't overwrite any other env vars
	// which potentially has the same name and to bring these closer to what
	// a (F)CGI server would proxy to a backend service
	// Replace hyphen with underscore and make them all upper case
	for key, values := range headers {
		h := "HTTP_" + strings.ReplaceAll(strings.ToUpper(key), "-", "_") + "=" + strings.Join(values, ",")
		// log.Printf("Adding header: %s", h)
		cmd.Env = append(cmd.Env, h)
	}

	pty, err := pty.Start(cmd)
	if err != nil {
		// todo close cmd?
		return nil, errors.Wrapf(err, "failed to start command `%s`", command)
	}
	ptyClosed := make(chan struct{})

	lcmd := &LocalCommand{
		command: command,
		argv:    argv,

		closeSignal:  DefaultCloseSignal,
		closeTimeout: DefaultCloseTimeout,

		cmd:       cmd,
		pty:       pty,
		ptyClosed: ptyClosed,
	}

	for _, option := range options {
		option(lcmd)
	}

	// When the process is closed by the user,
	// close pty so that Read() on the pty breaks with an EOF.
	go func() {
		defer func() {
			lcmd.ptyMu.Lock()
			_ = lcmd.pty.Close()
			lcmd.ptyMu.Unlock()
			close(lcmd.ptyClosed)
		}()

		_ = lcmd.cmd.Wait()
	}()

	return lcmd, nil
}

func (lcmd *LocalCommand) Read(p []byte) (n int, err error) {
	lcmd.ptyMu.Lock()
	defer lcmd.ptyMu.Unlock()
	return lcmd.pty.Read(p)
}

func (lcmd *LocalCommand) Write(p []byte) (n int, err error) {
	lcmd.ptyMu.Lock()
	defer lcmd.ptyMu.Unlock()
	return lcmd.pty.Write(p)
}

func (lcmd *LocalCommand) Close() error {
	if lcmd.cmd != nil && lcmd.cmd.Process != nil {
		_ = lcmd.cmd.Process.Signal(lcmd.closeSignal)
	}
	for {
		select {
		case <-lcmd.ptyClosed:
			return nil
		case <-lcmd.closeTimeoutC():
			_ = lcmd.cmd.Process.Signal(syscall.SIGKILL)
		}
	}
}

func (lcmd *LocalCommand) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{
		"command": lcmd.command,
		"argv":    lcmd.argv,
		"pid":     lcmd.cmd.Process.Pid,
	}
}

func (lcmd *LocalCommand) ResizeTerminal(width int, height int) error {
	lcmd.ptyMu.Lock()
	defer lcmd.ptyMu.Unlock()
	window := pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
		X:    0,
		Y:    0,
	}
	err := pty.Setsize(lcmd.pty, &window)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (lcmd *LocalCommand) closeTimeoutC() <-chan time.Time {
	if lcmd.closeTimeout >= 0 {
		return time.After(lcmd.closeTimeout)
	}

	return make(chan time.Time)
}
