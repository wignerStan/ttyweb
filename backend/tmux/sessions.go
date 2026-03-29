package tmux

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"ttyweb/pkg/validate"
)

// ListSessions returns all tmux sessions.
func ListSessions() ([]Session, error) {
	out, err := tmuxOutput("list-sessions", "-F", "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}")
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	if out == "" {
		return nil, nil
	}

	lines := strings.Split(out, "\n")
	sessions := make([]Session, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		s := Session{
			Name:     parts[0],
			Windows:  atoi(parts[1]),
			Created:  parts[2],
			Attached: parts[3] == "1",
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// ListPanes returns all panes in a session.
func ListPanes(session string) ([]Pane, error) {
	if err := validate.SessionName(session); err != nil {
		return nil, fmt.Errorf("list panes: %w", err)
	}
	out, err := tmuxOutput("list-panes", "-t", session,
		"-F", "#{pane_id}:#{window_index}:#{pane_title}:#{pane_current_command}:#{pane_dead}:#{pane_width}:#{pane_height}")
	if err != nil {
		return nil, fmt.Errorf("list panes %s: %w", session, err)
	}
	if out == "" {
		return nil, nil
	}

	lines := strings.Split(out, "\n")
	panes := make([]Pane, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}
		p := Pane{
			ID:      parts[0],
			Session: session,
			Window:  atoi(parts[1]),
			Title:   parts[2],
			Current: parts[3],
			Running: parts[4] == "0",
			Width:   atoi(parts[5]),
			Height:  atoi(parts[6]),
		}
		panes = append(panes, p)
	}
	return panes, nil
}

// GetSessionDetail returns a session with all its panes.
func GetSessionDetail(session string) (*SessionDetail, error) {
	if err := validate.SessionName(session); err != nil {
		return nil, fmt.Errorf("get session detail: %w", err)
	}
	panes, err := ListPanes(session)
	if err != nil {
		return nil, err
	}

	// Get session metadata
	sessions, err := ListSessions()
	if err != nil {
		return nil, err
	}

	var sess Session
	for _, s := range sessions {
		if s.Name == session {
			sess = s
			break
		}
	}
	if sess.Name == "" {
		return nil, fmt.Errorf("session %s not found", session)
	}

	return &SessionDetail{Session: sess, Panes: panes}, nil
}

// CreateSession creates a new detached tmux session.
func CreateSession(name string, command ...string) (string, error) {
	if name != "" {
		if err := validate.SessionName(name); err != nil {
			return "", fmt.Errorf("create session: %w", err)
		}
	}
	args := []string{"new-session", "-d"}
	if name != "" {
		args = append(args, "-s", name)
	}
	if len(command) > 0 {
		args = append(args, command...)
	} else {
		args = append(args, os.Getenv("SHELL"))
		if args[len(args)-1] == "" {
			args[len(args)-1] = "/bin/sh"
		}
	}
	_, err := tmuxExec(args...)
	if err != nil {
		return "", fmt.Errorf("create session %s: %w", name, err)
	}
	if name == "" {
		// Get the auto-generated name
		sessions, err := ListSessions()
		if err != nil || len(sessions) == 0 {
			return "", err
		}
		return sessions[len(sessions)-1].Name, nil
	}
	return name, nil
}

// KillSession destroys a tmux session.
func KillSession(name string) error {
	if err := validate.SessionName(name); err != nil {
		return fmt.Errorf("kill session: %w", err)
	}
	_, err := tmuxExec("kill-session", "-t", name)
	if err != nil {
		return fmt.Errorf("kill session %s: %w", name, err)
	}
	return nil
}

// SendKeys sends text/keys to a pane.
func SendKeys(pane string, keys ...string) error {
	args := make([]string, 0, 3+len(keys))
	args = append(args, "send-keys", "-t", pane)
	args = append(args, keys...)
	_, err := tmuxExec(args...)
	if err != nil {
		return fmt.Errorf("send-keys to %s: %w", pane, err)
	}
	return nil
}

// CapturePane captures the visible content of a pane.
func CapturePane(pane string, lines int) (string, error) {
	args := []string{"capture-pane", "-t", pane, "-p", "-e"}
	if lines > 0 {
		args = append(args, "-S", fmt.Sprintf("-%d", lines))
	}
	out, err := tmuxOutput(args...)
	if err != nil {
		return "", fmt.Errorf("capture pane %s: %w", pane, err)
	}
	return out, nil
}

// ResizePane resizes a pane.
func ResizePane(pane string, width, height int) error {
	if err := validate.PaneID(pane); err != nil {
		return fmt.Errorf("resize pane: %w", err)
	}
	_, err := tmuxExec("resize-pane", "-t", pane,
		"-x", strconv.Itoa(width), "-y", strconv.Itoa(height))
	if err != nil {
		return fmt.Errorf("resize pane %s: %w", pane, err)
	}
	return nil
}

// NewWindow creates a new window in a session and returns the pane ID.
func NewWindow(session string, name string) (string, error) {
	args := []string{"new-window", "-d", "-t", session}
	if name != "" {
		args = append(args, "-n", name)
	}
	_, err := tmuxExec(args...)
	if err != nil {
		return "", fmt.Errorf("new window in %s: %w", session, err)
	}
	return CurrentPane(session)
}

// CurrentPane returns the active pane ID in a session.
func CurrentPane(session string) (string, error) {
	out, err := tmuxOutput("display-message", "-p", "-t", session,
		"#{pane_id}")
	if err != nil {
		return "", fmt.Errorf("current pane %s: %w", session, err)
	}
	return strings.TrimSpace(out), nil
}

// KillPane kills a pane.
func KillPane(pane string) error {
	_, err := tmuxExec("kill-pane", "-t", pane)
	if err != nil {
		return fmt.Errorf("kill pane %s: %w", pane, err)
	}
	return nil
}

// SessionsJSON returns sessions as JSON (for REST API).
func SessionsJSON() ([]byte, error) {
	sessions, err := ListSessions()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(sessions)
	if err != nil {
		return nil, fmt.Errorf("SessionsJSON: marshal: %w", err)
	}
	return data, nil
}

// SessionDetailJSON returns a session with panes as JSON.
func SessionDetailJSON(name string) ([]byte, error) {
	detail, err := GetSessionDetail(name)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(detail)
	if err != nil {
		return nil, fmt.Errorf("SessionDetailJSON: marshal: %w", err)
	}
	return data, nil
}

// tmuxOutput runs a tmux command and returns stdout.
func tmuxOutput(args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "tmux", args...) //nolint:gosec // reason: hardcoded binary, args validated upstream
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux %s: %w", strings.Join(args, " "), err)
	}
	return out.String(), nil
}

// tmuxExec runs a tmux command (ignoring output).
func tmuxExec(args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "tmux", args...) //nolint:gosec // reason: hardcoded binary, args validated upstream
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// IsServerRunning checks if a tmux server is running.
func IsServerRunning() bool {
	_, err := tmuxOutput("has-session")
	return err == nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
