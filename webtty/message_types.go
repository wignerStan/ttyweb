package webtty

// Protocols defines the name of this protocol,
// which is supposed to be used to the subprotocol of Websockt streams.
var Protocols = []string{"webtty"}

const (
	// UnknownInput is an unknown message type, maybe sent by a bug.
	UnknownInput = '0'
	// Input represents user input, typically from a keyboard.
	Input = '1'
	// Ping represents a ping to the server.
	Ping = '2'
	// ResizeTerminal notifies that the browser size has been changed.
	ResizeTerminal = '3'
	// SetEncoding changes the character encoding.
	SetEncoding = '4'
)

const (
	// UnknownOutput is an unknown message type, maybe set by a bug.
	UnknownOutput = '0'
	// Output represents normal output to the terminal.
	Output = '1'
	// Pong represents a pong to the browser.
	Pong = '2'
	// SetWindowTitle sets the window title of the terminal.
	SetWindowTitle = '3'
	// SetPreferences sets terminal preferences.
	SetPreferences = '4'
	// SetReconnect makes the terminal reconnect.
	SetReconnect = '5'
	// SetBufferSize sets the input buffer size.
	SetBufferSize = '6'
	// SetMetadata sends side-channel metadata (AI state changes, notifications).
	SetMetadata = '7'
)
