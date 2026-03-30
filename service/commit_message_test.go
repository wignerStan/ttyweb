package service

import (
	"context"
	"strings"
	"testing"

	"ttyweb/config"
)

func TestGenerateCommitMessage_EmptyDiff(t *testing.T) {
	t.Parallel()

	svc := NewCommitMessageService(nil)
	_, err := svc.GenerateCommitMessage(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty diff")
	}
	if !strings.Contains(err.Error(), "empty diff") {
		t.Errorf("expected 'empty diff' in error, got %q", err.Error())
	}
}

func TestGenerateCommitMessage_WhitespaceDiff(t *testing.T) {
	t.Parallel()

	svc := NewCommitMessageService(nil)
	_, err := svc.GenerateCommitMessage(context.Background(), "   \n\t\n  ")
	if err == nil {
		t.Fatal("expected error for whitespace-only diff")
	}
}

func TestGenerateCommitMessage_NoAIClient(t *testing.T) {
	t.Parallel()

	// nil config means no API key, so client should be nil.
	svc := NewCommitMessageService(nil)
	_, err := svc.GenerateCommitMessage(context.Background(), "some diff content")
	if err == nil {
		t.Fatal("expected error when AI client is not configured")
	}
	if !strings.Contains(err.Error(), "AI client not configured") {
		t.Errorf("expected 'AI client not configured' in error, got %q", err.Error())
	}
}

func TestGenerateCommitMessage_NoAPIKey(t *testing.T) {
	t.Parallel()

	// Config with empty API key should produce a nil client.
	cfg := &config.Config{}
	svc := NewCommitMessageService(cfg)
	_, err := svc.GenerateCommitMessage(context.Background(), "some diff")
	if err == nil {
		t.Fatal("expected error when API key is empty")
	}
	if !strings.Contains(err.Error(), "AI client not configured") {
		t.Errorf("expected 'AI client not configured' in error, got %q", err.Error())
	}
}

func TestBuildCommitPrompt(t *testing.T) {
	t.Parallel()

	diff := "--- a/file.go\n+++ b/file.go\n@@ -1 +1 @@\n-old\n+new"
	prompt := buildCommitPrompt(diff)

	if !strings.Contains(prompt, "conventional commit") {
		t.Error("prompt should mention 'conventional commit'")
	}
	if !strings.Contains(prompt, diff) {
		t.Error("prompt should contain the diff")
	}
}
