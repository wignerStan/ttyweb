package ai

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEncodeProjectPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/home/user/projects/my-app", "-home-user-projects-my-app"},
		{"D:\\codes\\2025\\aicode-kanban", "D--codes-2025-aicode-kanban"},
		{"/home/user/game_system2/next", "-home-user-game_system2-next"},
		{"/tmp/test_project", "-tmp-test_project"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := EncodeProjectPath(tt.input)
			if result != tt.expected {
				t.Errorf("EncodeProjectPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEncodeProjectPath_NoCollisionBetweenUnderscoreAndDash(t *testing.T) {
	underscore := EncodeProjectPath("/my_project")
	dash := EncodeProjectPath("/my-project")

	if underscore == dash {
		t.Errorf("EncodeProjectPath produces collision: /my_project and /my-project both encode to %q", underscore)
	}
	if underscore != "-my_project" {
		t.Errorf("EncodeProjectPath(%q) = %q, want %q", "/my_project", underscore, "-my_project")
	}
	if dash != "-my-project" {
		t.Errorf("EncodeProjectPath(%q) = %q, want %q", "/my-project", dash, "-my-project")
	}
}

func TestExtractCodexSessionID(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{
			"rollout-2025-12-01T04-14-23-019ad666-f5ab-7501-a616-bbdc79da615b.jsonl",
			"019ad666-f5ab-7501-a616-bbdc79da615b",
		},
		{
			"rollout-2025-12-01T12-30-45-abcdef12-3456-7890-abcd-ef1234567890.jsonl",
			"abcdef12-3456-7890-abcd-ef1234567890",
		},
		{"invalid.jsonl", ""},
		{"not-a-rollout-file.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := extractCodexSessionID(tt.filename)
			if result != tt.expected {
				t.Errorf("extractCodexSessionID(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestScanClaudeProjects(t *testing.T) {
	homeDir := t.TempDir()
	claudeDir := filepath.Join(homeDir, ".claude", "projects")

	// Create some project directories.
	if err := os.MkdirAll(filepath.Join(claudeDir, "-home-user-project-a"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(claudeDir, "-home-user-project-b"), 0755); err != nil {
		t.Fatal(err)
	}
	// Create a file (should be skipped).
	if err := os.WriteFile(filepath.Join(claudeDir, "readme.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	// Override home directory by temporarily setting HOME.
	t.Setenv("HOME", homeDir)

	projects, err := ScanClaudeProjects()
	if err != nil {
		t.Fatalf("ScanClaudeProjects() error = %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("ScanClaudeProjects() returned %d projects, want 2", len(projects))
	}
}

func TestScanClaudeProjects_NonexistentDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	projects, err := ScanClaudeProjects()
	if err != nil {
		t.Fatalf("ScanClaudeProjects() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("ScanClaudeProjects() returned %d projects, want 0", len(projects))
	}
}

func TestScanClaudeSessions(t *testing.T) {
	homeDir := t.TempDir()
	claudeDir := filepath.Join(homeDir, ".claude", "projects")
	projectDir := filepath.Join(claudeDir, "-home-user-myproject")

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create session files.
	content := `{"type":"user","message":{"role":"user","content":"Hello!"},"timestamp":"2025-12-01T10:00:00Z","sessionId":"sess-1"}`
	if err := os.WriteFile(filepath.Join(projectDir, "sess-1.jsonl"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "sess-2.jsonl"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	// Agent file should be skipped.
	if err := os.WriteFile(filepath.Join(projectDir, "agent-sess-1.jsonl"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	// Non-JSONL file should be skipped.
	if err := os.WriteFile(filepath.Join(projectDir, "readme.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", homeDir)

	sessions, err := ScanClaudeSessions("/home/user/myproject")
	if err != nil {
		t.Fatalf("ScanClaudeSessions() error = %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("ScanClaudeSessions() returned %d sessions, want 2", len(sessions))
	}

	found := map[string]bool{}
	for _, s := range sessions {
		found[s.SessionID] = true
		if s.Type != string(AssistantTypeClaudeCode) {
			t.Errorf("session %s has wrong type: %s", s.SessionID, s.Type)
		}
		if s.ProjectPath != "/home/user/myproject" {
			t.Errorf("session %s has wrong project path: %s", s.SessionID, s.ProjectPath)
		}
	}

	if !found["sess-1"] || !found["sess-2"] {
		t.Errorf("missing session IDs in result: %v", found)
	}
}

func TestScanClaudeSessions_NonexistentProject(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessions, err := ScanClaudeSessions("/nonexistent/project")
	if err != nil {
		t.Fatalf("ScanClaudeSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("ScanClaudeSessions() returned %d sessions for nonexistent project, want 0", len(sessions))
	}
}

func TestScanCodexSessions(t *testing.T) {
	homeDir := t.TempDir()
	now := time.Now()
	dateDir := filepath.Join(homeDir, ".codex", "sessions",
		now.Format("2006"), now.Format("01"), now.Format("02"))

	if err := os.MkdirAll(dateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create rollout files.
	filename := "rollout-" + now.Format("2006-01-02T15-04-05") + "-019ad666-f5ab-7501-a616-bbdc79da615b.jsonl"
	content := `{"timestamp":"2025-11-30T20:14:23Z","type":"session_meta","payload":{"id":"019ad666-f5ab-7501-a616-bbdc79da615b"}}`
	if err := os.WriteFile(filepath.Join(dateDir, filename), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	// Non-rollout file should be skipped.
	if err := os.WriteFile(filepath.Join(dateDir, "other.jsonl"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", homeDir)

	sessions, err := ScanCodexSessions()
	if err != nil {
		t.Fatalf("ScanCodexSessions() error = %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("ScanCodexSessions() returned %d sessions, want 1", len(sessions))
	}

	if sessions[0].SessionID != "019ad666-f5ab-7501-a616-bbdc79da615b" {
		t.Errorf("session ID = %q, want %q", sessions[0].SessionID, "019ad666-f5ab-7501-a616-bbdc79da615b")
	}
	if sessions[0].Type != string(AssistantTypeCodex) {
		t.Errorf("session type = %q, want %q", sessions[0].Type, string(AssistantTypeCodex))
	}
}

func TestScanCodexSessions_NonexistentDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	sessions, err := ScanCodexSessions()
	if err != nil {
		t.Fatalf("ScanCodexSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("ScanCodexSessions() returned %d sessions, want 0", len(sessions))
	}
}
