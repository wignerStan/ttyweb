package ai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamChatCompletion_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer token")
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		messages := []string{
			`data: {"id":"1","object":"chat.completion.chunk","created":1234,"choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"1","object":"chat.completion.chunk","created":1234,"choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`data: [DONE]`,
		}
		for _, msg := range messages {
			_, _ = fmt.Fprintf(w, "%s\n\n", msg)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	var tokens []string
	err := StreamChatCompletion(
		context.Background(),
		srv.URL,
		"test-key",
		"gpt-4",
		"You are helpful.",
		"Say hello.",
		func(token string) {
			tokens = append(tokens, token)
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "Hello" {
		t.Errorf("expected 'Hello', got %q", tokens[0])
	}
	if tokens[1] != " world" {
		t.Errorf("expected ' world', got %q", tokens[1])
	}
}

func TestStreamChatCompletion_NonOKStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, "invalid api key")
	}))
	defer srv.Close()

	err := StreamChatCompletion(
		context.Background(),
		srv.URL,
		"bad-key",
		"gpt-4",
		"sys", "user",
		func(token string) {},
	)

	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestStreamChatCompletion_ContextCanceled(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"chunk\"}}]}\n\n")
		flusher.Flush()
		// Block to simulate slow stream.
		select {}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := StreamChatCompletion(ctx, srv.URL, "", "gpt-4", "", "", func(string) {})
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestStreamChatCompletion_SkipsNonDataLines(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		// Send non-data lines that should be skipped.
		_, _ = fmt.Fprintf(w, ":comment\n\n")
		_, _ = fmt.Fprintf(w, "event:message\n\n")
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"only\"}}]}\n\n")
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	var tokens []string
	err := StreamChatCompletion(
		context.Background(), srv.URL, "", "gpt-4", "", "",
		func(token string) { tokens = append(tokens, token) },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "only" {
		t.Fatalf("expected ['only'], got %v", tokens)
	}
}

func TestStreamChatCompletion_SkipsMalformedJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		_, _ = fmt.Fprintf(w, "data: {invalid json}\n\n")
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"valid\"}}]}\n\n")
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	var tokens []string
	err := StreamChatCompletion(
		context.Background(), srv.URL, "", "gpt-4", "", "",
		func(token string) { tokens = append(tokens, token) },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "valid" {
		t.Fatalf("expected ['valid'], got %v", tokens)
	}
}

func TestStreamChatCompletion_EmptyChoices(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		_, _ = fmt.Fprintf(w, "data: {\"choices\":[]}\n\n")
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	var tokens []string
	err := StreamChatCompletion(
		context.Background(), srv.URL, "", "gpt-4", "", "",
		func(token string) { tokens = append(tokens, token) },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("expected no tokens, got %v", tokens)
	}
}

func TestStreamChatCompletion_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "internal server error")
	}))
	defer srv.Close()

	err := StreamChatCompletion(
		context.Background(), srv.URL, "", "gpt-4", "", "",
		func(token string) {},
	)
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}
