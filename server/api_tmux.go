package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
)

// handleTmuxConfig returns the tmux prefix key configuration.
func (server *Server) handleTmuxConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeAPISuccess(w, map[string]interface{}{
		"code":  "\u0002",
		"label": "Ctrl+B",
	})
}

// handleQuickDirs returns quick-access directories (stub).
func (server *Server) handleQuickDirs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeAPISuccess(w, map[string]interface{}{
		"dirs": []interface{}{},
	})
}

// handleTmuxNewWindow creates a new tmux window in a session.
func (server *Server) handleTmuxNewWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if server.factory.Name() != "tmux" {
		writeAPIError(w, http.StatusServiceUnavailable, "tmux backend not active")
		return
	}

	var body struct {
		Session string `json:"session"`
		Dir     string `json:"dir"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Session == "" {
		writeAPIError(w, http.StatusBadRequest, "session is required")
		return
	}

	args := []string{"new-window", "-d", "-t", body.Session}
	if body.Name != "" {
		args = append(args, "-n", body.Name)
	}
	if body.Dir != "" {
		args = append(args, "-c", body.Dir)
	}

	cmd := exec.CommandContext(r.Context(), "tmux", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create window: "+string(output))
		return
	}

	writeAPISuccess(w, map[string]string{
		"session": body.Session,
		"status":  "created",
	})
}

// handleTmuxNewSession creates a new tmux session using the sessionManager.
func (server *Server) handleTmuxNewSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sm := server.sessionManager()
	if !sm.IsAvailable() {
		writeAPIError(w, http.StatusServiceUnavailable, "session management not available")
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name, err := sm.CreateSession(body.Name)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create session: "+err.Error())
		return
	}

	writeAPISuccess(w, map[string]string{
		"name":   name,
		"status": "created",
	})
}

// tmuxTreeSession represents a session in the tree response.
type tmuxTreeSession struct {
	SessionName string           `json:"sessionName"`
	SessionID   string           `json:"sessionId"`
	Windows     []tmuxTreeWindow `json:"windows"`
}

// tmuxTreeWindow represents a window in the tree response.
type tmuxTreeWindow struct {
	WindowIndex int            `json:"windowIndex"`
	WindowName  string         `json:"windowName"`
	WindowID    string         `json:"windowId"`
	Panes       []tmuxTreePane `json:"panes"`
}

// tmuxTreePane represents a pane in the tree response.
type tmuxTreePane struct {
	PaneID      string `json:"paneId"`
	PaneTitle   string `json:"paneTitle"`
	PaneCommand string `json:"paneCommand"`
}

// handleTmuxTree builds a nested tree of sessions, windows, and panes.
func (server *Server) handleTmuxTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sm := server.sessionManager()
	if !sm.IsAvailable() {
		writeAPISuccess(w, []interface{}{})
		return
	}

	// Get all sessions.
	sessionsRaw, err := sm.ListSessions()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list sessions: "+err.Error())
		return
	}

	// Parse sessions list.
	var sessions []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(sessionsRaw, &sessions); err != nil {
		writeAPISuccessRaw(w, sessionsRaw)
		return
	}

	tree := make([]tmuxTreeSession, 0, len(sessions))
	for _, sess := range sessions {
		// Get session detail (includes panes).
		detailRaw, err := sm.GetSessionDetail(sess.Name)
		if err != nil {
			continue
		}

		var detail struct {
			Panes []struct {
				ID      string `json:"id"`
				Window  int    `json:"window"`
				Title   string `json:"title"`
				Current string `json:"current_command"`
			} `json:"panes"`
		}
		if err := json.Unmarshal(detailRaw, &detail); err != nil {
			continue
		}

		// Group panes by window index.
		windowsMap := make(map[int]*tmuxTreeWindow)
		var windowOrder []int
		for _, pane := range detail.Panes {
			if _, exists := windowsMap[pane.Window]; !exists {
				windowsMap[pane.Window] = &tmuxTreeWindow{
					WindowIndex: pane.Window,
					WindowName:  "",
					WindowID:    fmt.Sprintf("@%d", pane.Window),
				}
				windowOrder = append(windowOrder, pane.Window)
			}
			windowsMap[pane.Window].Panes = append(windowsMap[pane.Window].Panes, tmuxTreePane{
				PaneID:      pane.ID,
				PaneTitle:   pane.Title,
				PaneCommand: pane.Current,
			})
		}

		// Build windows in order.
		var windows []tmuxTreeWindow
		for _, idx := range windowOrder {
			windows = append(windows, *windowsMap[idx])
		}

		tree = append(tree, tmuxTreeSession{
			SessionName: sess.Name,
			SessionID:   fmt.Sprintf("$%s", sess.Name),
			Windows:     windows,
		})
	}

	if tree == nil {
		tree = []tmuxTreeSession{}
	}

	writeAPISuccess(w, map[string]interface{}{
		"sessions": tree,
	})
}

// handleTmuxSendKeys sends keys to a specific tmux pane using exec.Command.
func (server *Server) handleTmuxSendKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if server.factory.Name() != "tmux" {
		writeAPIError(w, http.StatusServiceUnavailable, "tmux backend not active")
		return
	}

	var body struct {
		Pane string `json:"pane"`
		Keys string `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Pane == "" {
		writeAPIError(w, http.StatusBadRequest, "pane is required")
		return
	}
	if body.Keys == "" {
		writeAPIError(w, http.StatusBadRequest, "keys is required")
		return
	}

	args := []string{"send-keys", "-t", body.Pane, body.Keys}
	cmd := exec.CommandContext(r.Context(), "tmux", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to send keys: "+string(output))
		return
	}

	writeAPISuccess(w, map[string]string{
		"pane":   body.Pane,
		"status": "sent",
	})
}

// handleTmuxPaneMode returns the current pane mode (stub).
func (server *Server) handleTmuxPaneMode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeAPISuccess(w, map[string]string{
		"mode": "pane",
	})
}
