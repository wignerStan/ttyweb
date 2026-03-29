package tmux

// Session represents a tmux session.
type Session struct {
	Name     string `json:"name"`
	Windows  int    `json:"windows"`
	Created  string `json:"created,omitempty"`
	Attached bool   `json:"attached"`
}

// Pane represents a tmux pane within a session.
type Pane struct {
	ID      string `json:"id"`
	Session string `json:"session"`
	Window  int    `json:"window"`
	Title   string `json:"title"`
	Current string `json:"current_command"`
	Running bool   `json:"running"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// SessionDetail is a session with its panes.
type SessionDetail struct {
	Session
	Panes []Pane `json:"panes"`
}
