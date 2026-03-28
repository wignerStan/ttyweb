package server

import (
	"ttyweb/backend"
)

// Slave is a PTY slave with a Close method.
type Slave = backend.Slave

// Factory creates Slave instances for each connection.
type Factory = backend.Factory
