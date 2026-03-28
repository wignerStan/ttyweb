package localcommand

import (
	"testing"
)

func TestWindowTitleVariables(t *testing.T) {
	tests := []struct {
		name    string
		command string
		argv    []string
	}{
		{
			name:    "command key present with simple command",
			command: "/bin/bash",
			argv:    []string{},
		},
		{
			name:    "command key present with arguments",
			command: "/bin/ls",
			argv:    []string{"-la", "/tmp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lcmd, err := New(tt.command, tt.argv, nil)
			if err != nil {
				t.Skipf("skipping: PTY not available: %v", err)
			}
			defer lcmd.Close()

			vars := lcmd.WindowTitleVariables()

			if vars == nil {
				t.Fatal("WindowTitleVariables() returned nil")
			}

			cmdVal, ok := vars["command"]
			if !ok {
				t.Error("WindowTitleVariables() missing 'command' key")
			}
			if cmdVal != tt.command {
				t.Errorf("command = %v, want %v", cmdVal, tt.command)
			}

			argvVal, ok := vars["argv"]
			if !ok {
				t.Error("WindowTitleVariables() missing 'argv' key")
			}
			if len(argvVal.([]string)) != len(tt.argv) {
				t.Errorf("argv length = %d, want %d", len(argvVal.([]string)), len(tt.argv))
			}

			pidVal, ok := vars["pid"]
			if !ok {
				t.Error("WindowTitleVariables() missing 'pid' key")
			}
			if pidVal.(int) <= 0 {
				t.Errorf("pid = %d, expected positive value", pidVal)
			}
		})
	}
}

func TestResizeTerminal(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{
			name:   "standard size 80x24",
			width:  80,
			height: 24,
		},
		{
			name:   "large size 200x50",
			width:  200,
			height: 50,
		},
		{
			name:   "small size 40x10",
			width:  40,
			height: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lcmd, err := New("/bin/cat", []string{}, nil)
			if err != nil {
				t.Skipf("skipping: PTY not available: %v", err)
			}
			defer lcmd.Close()

			err = lcmd.ResizeTerminal(tt.width, tt.height)
			if err != nil {
				t.Errorf("ResizeTerminal(%d, %d) returned error: %v", tt.width, tt.height, err)
			}
		})
	}
}
