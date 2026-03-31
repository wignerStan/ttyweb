package server

import (
	"context"
	"testing"

	"ttyweb/ai"
	"ttyweb/webtty"

	"github.com/pkg/errors"
)

// mockTitleInterceptor is a test OutputInterceptor that returns predefined metadata.
type mockTitleInterceptor struct {
	metadata []ai.Metadata
}

func (m *mockTitleInterceptor) Intercept(_ []byte) []ai.Metadata { return m.metadata }

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

// TestAIStateMachineBridge_Intercept tests the bridge's Intercept method.
func TestAIStateMachineBridge_Intercept(t *testing.T) {
	sm := ai.NewStateMachine()
	inner := &mockTitleInterceptor{}

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

// TestTitleVariables_MissingName covers the error path when a variable unit name
// is not found in varUnits.
func TestTitleVariables_MissingName(t *testing.T) {
	srv := &Server{}
	_, err := srv.titleVariables(
		[]string{"server", "missing"},
		map[string]map[string]any{
			"server": {"key": "val"},
		},
	)
	if err == nil {
		t.Fatal("expected error for missing variable unit name")
	}
}

// TestTitleVariables_Success covers the successful path with multiple units.
func TestTitleVariables_Success(t *testing.T) {
	srv := &Server{}
	vars, err := srv.titleVariables(
		[]string{"server", "master", "slave"},
		map[string]map[string]any{
			"server": {"Version": "1.0"},
			"master": {"remote_addr": "10.0.0.1"},
			"slave":  {"command": "bash"},
		},
	)
	if err != nil {
		t.Fatalf("titleVariables failed: %v", err)
	}

	if vars["Version"] != "1.0" {
		t.Errorf("expected Version=1.0, got %v", vars["Version"])
	}
	if vars["remote_addr"] != "10.0.0.1" {
		t.Errorf("expected remote_addr=10.0.0.1, got %v", vars["remote_addr"])
	}
	if vars["command"] != "bash" {
		t.Errorf("expected command=bash, got %v", vars["command"])
	}

	// Verify safe net: later names override earlier keys.
	if vars["server"] == nil {
		t.Error("expected 'server' safe net key to be set")
	}
}

// TestRegisterAIStateChangeHandlers covers the state machine event registration.
func TestRegisterAIStateChangeHandlers(t *testing.T) {
	sm := ai.NewStateMachine()
	registerAIStateChangeHandlers(sm)

	// Trigger working transition — should not panic.
	sm.Transition("test-pane", ai.AIStateWorking)
	sm.Transition("test-pane", ai.AIStateIdle)
}

// TestWebttyOptions covers the webttyOptions method.
func TestWebttyOptions(t *testing.T) {
	srv := &Server{options: &Options{}}

	opts := srv.webttyOptions([]byte("title"))
	if len(opts) < 1 {
		t.Fatal("expected at least 1 option")
	}

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
