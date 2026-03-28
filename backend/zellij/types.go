package zellij

// Session represents a zellij session.
type Session struct {
	Name     string `json:"name"`
	Attached bool   `json:"attached"`
}

// Pane represents a zellij pane within a session.
type Pane struct {
	ID      string `json:"id"`
	Session string `json:"session"`
	Title   string `json:"title"`
	Running bool   `json:"running"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// SessionDetail is a session with its panes.
type SessionDetail struct {
	Session
	Panes []Pane `json:"panes"`
}
