package zellij

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"ttyweb/pkg/validate"
)

// ListSessions returns all zellij sessions.
func ListSessions() ([]Session, error) {
	out, err := zellijOutput("list-sessions")
	if err != nil {
		if strings.Contains(err.Error(), "no sessions") || strings.Contains(out, "no sessions") {
			return nil, nil
		}
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	if out == "" {
		return nil, nil
	}

	// zellij list-sessions outputs one session name per line
	sessions := make([]Session, 0)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		sessions = append(sessions, Session{
			Name:     line,
			Attached: false, // zellij doesn't expose this easily in list output
		})
	}
	return sessions, nil
}

// CreateSession creates a new detached zellij session.
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
		args = append(args, "--", command[0])
		args = append(args, command[1:]...)
	}
	_, err := zellijExec(args...)
	if err != nil {
		return "", fmt.Errorf("create session %s: %w", name, err)
	}
	if name == "" {
		sessions, err := ListSessions()
		if err != nil || len(sessions) == 0 {
			return "", err
		}
		return sessions[len(sessions)-1].Name, nil
	}
	return name, nil
}

// KillSession destroys a zellij session.
func KillSession(name string) error {
	if err := validate.SessionName(name); err != nil {
		return fmt.Errorf("kill session: %w", err)
	}
	_, err := zellijExec("kill-session", name)
	if err != nil {
		return fmt.Errorf("kill session %s: %w", name, err)
	}
	return nil
}

// IsServerRunning checks if zellij is available.
func IsServerRunning() bool {
	_, err := exec.LookPath("zellij")
	return err == nil
}

// SessionsJSON returns sessions as JSON.
func SessionsJSON() ([]byte, error) {
	sessions, err := ListSessions()
	if err != nil {
		return nil, err
	}
	return json.Marshal(sessions)
}

// zellijOutput runs a zellij command and returns stdout.
func zellijOutput(args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "zellij", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}

// zellijExec runs a zellij command (ignoring output).
func zellijExec(args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "zellij", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
