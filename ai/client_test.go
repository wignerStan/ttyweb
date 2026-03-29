package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("key", "https://api.example.com", "gpt-4")
	if c.apiKey != "key" {
		t.Errorf("expected apiKey key, got %q", c.apiKey)
	}
	if c.apiURL != "https://api.example.com" {
		t.Errorf("expected apiURL https://api.example.com, got %q", c.apiURL)
	}
	if c.model != "gpt-4" {
		t.Errorf("expected model gpt-4, got %q", c.model)
	}
	if c.client == nil {
		t.Error("expected non-nil http client")
	}
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	c := NewClient("key", "https://api.example.com/", "gpt-4")
	if c.apiURL != "https://api.example.com" {
		t.Errorf("expected trailing slash to be trimmed, got %q", c.apiURL)
	}
}

func TestChatCompletion_Success(t *testing.T) {
	expected := "ls -la"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request.
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization Bearer test-key, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", r.Header.Get("Content-Type"))
		}

		var reqBody chatRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if reqBody.Model != "test-model" {
			t.Errorf("expected model test-model, got %q", reqBody.Model)
		}
		if len(reqBody.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(reqBody.Messages))
		}
		if reqBody.Messages[0].Role != "system" {
			t.Errorf("expected first message role system, got %q", reqBody.Messages[0].Role)
		}
		if reqBody.Messages[1].Role != "user" {
			t.Errorf("expected second message role user, got %q", reqBody.Messages[1].Role)
		}

		resp := chatResponse{
			Choices: []chatChoice{
				{
					Message: chatMessage{
						Role:    "assistant",
						Content: expected,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-key", srv.URL, "test-model")
	result, err := client.ChatCompletion(context.Background(), "You are helpful.", "list files")
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestChatCompletion_EmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{
			Choices: []chatChoice{
				{
					Message: chatMessage{
						Role:    "assistant",
						Content: "",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-key", srv.URL, "test-model")
	_, err := client.ChatCompletion(context.Background(), "sys", "user")
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
}

func TestChatCompletion_NoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{Choices: nil}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-key", srv.URL, "test-model")
	_, err := client.ChatCompletion(context.Background(), "sys", "user")
	if err == nil {
		t.Fatal("expected error for no choices, got nil")
	}
}

func TestChatCompletion_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "invalid API key"},
		})
	}))
	defer srv.Close()

	client := NewClient("bad-key", srv.URL, "test-model")
	_, err := client.ChatCompletion(context.Background(), "sys", "user")
	if err == nil {
		t.Fatal("expected error for unauthorized, got nil")
	}
}

func TestChatCompletion_ErrorField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{
			Error: &chatError{Message: "rate limit exceeded"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-key", srv.URL, "test-model")
	_, err := client.ChatCompletion(context.Background(), "sys", "user")
	if err == nil {
		t.Fatal("expected error for API error field, got nil")
	}
}

func TestChatCompletion_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		_ = json.NewEncoder(w).Encode(chatResponse{})
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	client := NewClient("test-key", srv.URL, "test-model")
	_, err := client.ChatCompletion(ctx, "sys", "user")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestChatCompletion_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := NewClient("test-key", srv.URL, "test-model")
	_, err := client.ChatCompletion(context.Background(), "sys", "user")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestExtractCommand_PlainText(t *testing.T) {
	result := ExtractCommand("ls -la")
	if result != "ls -la" {
		t.Errorf("expected 'ls -la', got %q", result)
	}
}

func TestExtractCommand_CodeFence(t *testing.T) {
	input := "Here is the command:\n```bash\nls -la /home\n```\nThat should work."
	result := ExtractCommand(input)
	if result != "ls -la /home" {
		t.Errorf("expected 'ls -la /home', got %q", result)
	}
}

func TestExtractCommand_CodeFenceNoLanguage(t *testing.T) {
	input := "```\nls -la\n```"
	result := ExtractCommand(input)
	if result != "ls -la" {
		t.Errorf("expected 'ls -la', got %q", result)
	}
}

func TestExtractCommand_CodeFenceNoNewline(t *testing.T) {
	// Opening fence without a newline (no code body) returns content after fence.
	input := "```text"
	result := ExtractCommand(input)
	if result != "text" {
		t.Errorf("expected 'text', got %q", result)
	}
}

func TestExtractCommand_CodeFenceMultiLine(t *testing.T) {
	input := "```\nfind . -name '*.go' -exec grep -l 'TODO' {} +\n&& echo 'Done'\n```"
	result := ExtractCommand(input)
	expected := "find . -name '*.go' -exec grep -l 'TODO' {} +\n&& echo 'Done'"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractCommand_NoClosingFence(t *testing.T) {
	input := "```bash\nls -la"
	result := ExtractCommand(input)
	if result != "ls -la" {
		t.Errorf("expected 'ls -la', got %q", result)
	}
}

func TestExtractCommand_Empty(t *testing.T) {
	result := ExtractCommand("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestExtractCommand_FirstFenceOnly(t *testing.T) {
	input := "```bash\necho first\n```\n```bash\necho second\n```"
	result := ExtractCommand(input)
	if result != "echo first" {
		t.Errorf("expected 'echo first', got %q", result)
	}
}
