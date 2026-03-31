package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ttyweb/ai"
	"ttyweb/config"
)

func TestBuildCommitPrompt_LargeDiff(t *testing.T) {
	largeDiff := strings.Repeat("a", 10000)
	prompt := buildCommitPrompt(largeDiff)
	if !strings.Contains(prompt, largeDiff) {
		t.Error("expected prompt to contain the full diff")
	}
}

func TestBuildCommitPrompt_EmptyDiff(t *testing.T) {
	prompt := buildCommitPrompt("")
	if prompt == "" {
		t.Error("expected non-empty prompt even with empty diff")
	}
	if !strings.Contains(prompt, "Generate a conventional commit message") {
		t.Error("expected prompt to contain instructions")
	}
}

func TestNewCommitMessageService_WithAPIKey(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIKey: "test-key",
			APIURL: "https://api.example.com/v1",
			Model:  "gpt-4",
		},
	}
	svc := NewCommitMessageService(cfg)
	if svc == nil {
		t.Error("expected non-nil service")
	}
}

func TestGenerateCommitMessage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": "feat: add new feature",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIKey: "test-key",
			APIURL: srv.URL,
			Model:  "gpt-4",
		},
	}
	svc := NewCommitMessageService(cfg)
	msg, err := svc.GenerateCommitMessage(context.Background(), "diff content here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: add new feature" {
		t.Errorf("expected 'feat: add new feature', got %q", msg)
	}
}

func TestGenerateCommitMessage_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIKey: "test-key",
			APIURL: srv.URL,
			Model:  "gpt-4",
		},
	}
	svc := NewCommitMessageService(cfg)
	_, err := svc.GenerateCommitMessage(context.Background(), "some diff")
	if err == nil {
		t.Error("expected error from API failure")
	}
}

func TestCommitMessageService_ClientCreated(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIKey: "key-123",
			APIURL: "https://api.openai.com/v1/chat/completions",
			Model:  "gpt-4o",
		},
	}
	svc := NewCommitMessageService(cfg)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	// Verify client was configured (no "not configured" error).
	_, err := svc.GenerateCommitMessage(context.Background(), "test diff")
	if err == nil {
		t.Error("expected error (no server running)")
	}
	if strings.Contains(err.Error(), "not configured") {
		t.Error("client should have been configured")
	}
}

// Verify the ai.Client type compiles correctly in this context.
var _ ai.Client
