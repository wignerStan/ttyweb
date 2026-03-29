package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AISession represents a detected AI assistant session.
type AISession struct {
	SessionID             string     `json:"sessionId"`
	Type                  string     `json:"type"`
	ProjectPath           string     `json:"projectPath,omitempty"`
	FilePath              string     `json:"filePath"`
	Model                 string     `json:"model,omitempty"`
	Title                 string     `json:"title,omitempty"`
	SessionStartedAt      time.Time  `json:"sessionStartedAt"`
	LastMessageAt         *time.Time `json:"lastMessageAt,omitempty"`
	MessageCount          int        `json:"messageCount"`
	AssistantMessageCount int        `json:"assistantMessageCount"`
	FileModTime           time.Time  `json:"fileModTime"`
	FileSize              int64      `json:"fileSize"`
}

// EncodeProjectPath converts a filesystem path to Claude Code's directory naming convention.
// Colons, slashes, and non-ASCII characters are replaced with dashes.
// Underscores are preserved to avoid collisions (e.g., /my_project vs /my-project).
func EncodeProjectPath(path string) string {
	path = filepath.Clean(path)
	// Explicitly convert backslashes to forward slashes (works on all OSes,
	// since filepath.ToSlash only converts OS-native separators).
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimRight(path, "/\\")

	var result strings.Builder
	for _, r := range path {
		switch {
		case r == ':' || r == '/':
			result.WriteRune('-')
		case r > 127:
			result.WriteRune('-')
		default:
			result.WriteRune(r)
		}
	}

	return result.String()
}

// homeSubdir returns an absolute path under the user's home directory.
func homeSubdir(relPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	return filepath.Join(home, relPath), nil
}

// claudeProjectsDir returns the path to ~/.claude/projects/.
func claudeProjectsDir() (string, error) {
	return homeSubdir(".claude/projects")
}

// codexSessionsDir returns the path to ~/.codex/sessions/.
func codexSessionsDir() (string, error) {
	return homeSubdir(".codex/sessions")
}

// ScanClaudeProjects lists project directories under ~/.claude/projects/.
// Returns decoded project paths (best-effort reversal of EncodeProjectPath).
func ScanClaudeProjects() ([]string, error) {
	dir, err := claudeProjectsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read claude projects directory: %w", err)
	}

	var projects []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projects = append(projects, entry.Name())
	}
	sort.Strings(projects)
	return projects, nil
}

// ScanClaudeSessions scans JSONL session files in ~/.claude/projects/<encoded-path>/.
// Returns parsed AISession metadata for each session file found.
func ScanClaudeSessions(projectPath string) ([]AISession, error) {
	projectsDir, err := claudeProjectsDir()
	if err != nil {
		return nil, err
	}

	encodedPath := EncodeProjectPath(projectPath)
	sessionDir := filepath.Join(projectsDir, encodedPath)

	entries, err := os.ReadDir(sessionDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory %q: %w", sessionDir, err)
	}

	var sessions []AISession
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip agent files and non-JSONL files.
		if strings.HasPrefix(name, "agent-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		filePath := filepath.Join(sessionDir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		sessionID := strings.TrimSuffix(name, ".jsonl")
		session := AISession{
			SessionID:   sessionID,
			Type:        string(AssistantTypeClaudeCode),
			ProjectPath: projectPath,
			FilePath:    filePath,
			FileModTime: info.ModTime(),
			FileSize:    info.Size(),
		}
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].FileModTime.After(sessions[j].FileModTime)
	})

	return sessions, nil
}

// ScanCodexSessions scans session files under ~/.codex/sessions/<year>/<month>/<day>/.
// Returns parsed AISession metadata for each rollout file found.
func ScanCodexSessions() ([]AISession, error) {
	sessionDir, err := codexSessionsDir()
	if err != nil {
		return nil, err
	}

	// Only scan today's directory for immediate results.
	now := time.Now()
	todayDir := filepath.Join(sessionDir, now.Format("2006"), now.Format("01"), now.Format("02"))

	todayEntries, err := os.ReadDir(todayDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, nil
	}

	var sessions []AISession
	for _, entry := range todayEntries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		filePath := filepath.Join(todayDir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		sessionID := extractCodexSessionID(name)
		if sessionID == "" {
			continue
		}

		session := AISession{
			SessionID:   sessionID,
			Type:        string(AssistantTypeCodex),
			FilePath:    filePath,
			FileModTime: info.ModTime(),
			FileSize:    info.Size(),
		}
		sessions = append(sessions, session)
	}

	// Sort by modification time (newest first).
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].FileModTime.After(sessions[j].FileModTime)
	})

	return sessions, nil
}

// extractCodexSessionID extracts the UUID from a Codex rollout filename.
// Format: rollout-2025-12-01T04-14-23-019ad666-f5ab-7501-a616-bbdc79da615b.jsonl
func extractCodexSessionID(filename string) string {
	name := strings.TrimPrefix(filename, "rollout-")
	name = strings.TrimSuffix(name, ".jsonl")

	parts := strings.Split(name, "-")
	if len(parts) < 9 {
		return ""
	}

	uuidParts := parts[len(parts)-5:]
	return strings.Join(uuidParts, "-")
}
