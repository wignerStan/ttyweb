package server

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"ttyweb/ai"
	"ttyweb/webtty"
)

// TestClassifyWSCloseError covers all branches of classifyWSCloseError.
func TestClassifyWSCloseError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		backendName  string
		wantContains string
	}{
		{"nil error", nil, "local", "normal close"},
		{"context canceled", context.Canceled, "local", "cancelation"},
		{"slave closed", webtty.ErrSlaveClosed, "tmux", "tmux"},
		{"master closed", webtty.ErrMasterClosed, "tmux", "client"},
		{"generic error", errors.New("something broke"), "local", "an error: something broke"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyWSCloseError(tc.err, tc.backendName)
			if got != tc.wantContains {
				t.Errorf("classifyWSCloseError() = %q, want %q", got, tc.wantContains)
			}
		})
	}
}

// TestWebttyOptions covers the webttyOptions method.
func TestWebttyOptions(t *testing.T) {
	srv := &Server{options: &Options{}}

	// Default options.
	opts := srv.webttyOptions([]byte("title"))
	if len(opts) < 1 {
		t.Fatal("expected at least 1 option")
	}

	// With all options enabled.
	srv.options.PermitWrite = true
	srv.options.EnableReconnect = true
	srv.options.ReconnectTime = 5
	srv.options.Width = 80
	srv.options.Height = 24

	opts = srv.webttyOptions([]byte("title"))
	if len(opts) < 5 {
		t.Errorf("expected at least 5 options with all enabled, got %d", len(opts))
	}
}

// mockInterceptor is a test OutputInterceptor that returns predefined metadata.
type mockInterceptor struct {
	metadata []ai.Metadata
}

func (m *mockInterceptor) Intercept(data []byte) []ai.Metadata {
	return m.metadata
}

// TestAIStateMachineBridge_Intercept tests the bridge's Intercept method.
func TestAIStateMachineBridge_Intercept(t *testing.T) {
	sm := ai.NewStateMachine()
	inner := &mockInterceptor{}

	bridge := &aiStateMachineBridge{
		inner:   inner,
		sm:      sm,
		paneKey: "pane-1",
	}

	// Test with no metadata.
	inner.metadata = nil
	result := bridge.Intercept([]byte("data"))
	if len(result) != 0 {
		t.Errorf("expected no metadata, got %d", len(result))
	}

	// Test with non-ai_state_change metadata.
	inner.metadata = []ai.Metadata{
		{Type: "other", Data: map[string]any{"key": "val"}},
	}
	result = bridge.Intercept([]byte("data"))
	if len(result) != 1 {
		t.Errorf("expected 1 metadata, got %d", len(result))
	}

	// Test with ai_state_change metadata.
	inner.metadata = []ai.Metadata{
		{Type: "ai_state_change", Data: map[string]any{"state": "working"}},
	}
	result = bridge.Intercept([]byte("data"))
	if len(result) != 1 {
		t.Errorf("expected 1 metadata, got %d", len(result))
	}

	// Test with ai_state_change but non-string state.
	inner.metadata = []ai.Metadata{
		{Type: "ai_state_change", Data: map[string]any{"state": 123}},
	}
	result = bridge.Intercept([]byte("data"))
	if len(result) != 1 {
		t.Errorf("expected 1 metadata, got %d", len(result))
	}
}
