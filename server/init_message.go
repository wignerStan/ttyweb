package server

// InitMessage is sent to the client on WebSocket connection.
type InitMessage struct {
	Arguments string `json:"Arguments,omitempty"`
	AuthToken string `json:"AuthToken,omitempty"`
}
