package localcommand

import (
	"syscall"
	"time"
)

// Option configures a LocalCommand.
type Option func(*LocalCommand)

// WithCloseSignal sets the signal used to close the command process.
func WithCloseSignal(signal syscall.Signal) Option {
	return func(lcmd *LocalCommand) {
		lcmd.closeSignal = signal
	}
}

// WithCloseTimeout sets the duration before force-killing the process.
func WithCloseTimeout(timeout time.Duration) Option {
	return func(lcmd *LocalCommand) {
		lcmd.closeTimeout = timeout
	}
}
