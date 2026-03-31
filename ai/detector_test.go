package ai

import "testing"

func TestDetectAssistant_ClaudeCode(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    AssistantType
		found   bool
	}{
		{
			name:    "claude code npm package",
			command: "node /home/user/.npm-global/lib/node_modules/@anthropic-ai/claude-code/cli.js",
			want:    AssistantTypeClaudeCode,
			found:   true,
		},
		{
			name:    "claude code bin path",
			command: "node /usr/local/lib/node_modules/@anthropic-ai/claude-code/bin/claude.js",
			want:    AssistantTypeClaudeCode,
			found:   true,
		},
		{
			name:    "claude binary",
			command: "/usr/local/bin/claude",
			want:    AssistantTypeClaudeCode,
			found:   true,
		},
		{
			name:    "claude code with args (full path)",
			command: "/usr/local/bin/claude -p \"write tests\"",
			want:    AssistantTypeClaudeCode,
			found:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DetectAssistant(tt.command)
			if ok != tt.found {
				t.Errorf("DetectAssistant(%q) found = %v, want %v", tt.command, ok, tt.found)
			}
			if ok && got != tt.want {
				t.Errorf("DetectAssistant(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

func TestDetectAssistant_Codex(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    AssistantType
		found   bool
	}{
		{
			name:    "codex npm package",
			command: "node /home/user/.npm-global/lib/node_modules/@openai/codex/cli.js",
			want:    AssistantTypeCodex,
			found:   true,
		},
		{
			name:    "codex bin path",
			command: "node /home/user/.npm-global/lib/node_modules/codex/bin/codex.js",
			want:    AssistantTypeCodex,
			found:   true,
		},
		{
			name:    "codex binary",
			command: "/usr/local/bin/codex",
			want:    AssistantTypeCodex,
			found:   true,
		},
		{
			name:    "codex with args (full path)",
			command: "/usr/local/bin/codex \"fix the bug in auth.go\"",
			want:    AssistantTypeCodex,
			found:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DetectAssistant(tt.command)
			if ok != tt.found {
				t.Errorf("DetectAssistant(%q) found = %v, want %v", tt.command, ok, tt.found)
			}
			if ok && got != tt.want {
				t.Errorf("DetectAssistant(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

func TestDetectAssistant_NotFound(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "regular bash command",
			command: "/bin/bash",
		},
		{
			name:    "vim editor",
			command: "vim /tmp/file.txt",
		},
		{
			name:    "empty string",
			command: "",
		},
		{
			name:    "python script",
			command: "python3 main.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := DetectAssistant(tt.command)
			if ok {
				t.Errorf("DetectAssistant(%q) should not detect any assistant", tt.command)
			}
		})
	}
}

func TestDetectAssistant_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    AssistantType
	}{
		{
			name:    "uppercase CLAUDE-CODE",
			command: "NODE /PATH/CLAUDE-CODE/CLI.JS",
			want:    AssistantTypeClaudeCode,
		},
		{
			name:    "mixed case Codex",
			command: "node /path/Codex/bin/codex.js",
			want:    AssistantTypeCodex,
		},
		{
			name:    "Windows backslashes",
			command: "node C:\\Users\\test\\AppData\\Roaming\\npm\\node_modules\\@anthropic-ai\\claude-code\\cli.js",
			want:    AssistantTypeClaudeCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DetectAssistant(tt.command)
			if !ok {
				t.Errorf("DetectAssistant(%q) should detect assistant", tt.command)
			}
			if got != tt.want {
				t.Errorf("DetectAssistant(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

func TestAssistantType_DisplayName(t *testing.T) {
	tests := []struct {
		atype AssistantType
		want  string
	}{
		{AssistantTypeClaudeCode, "Claude Code"},
		{AssistantTypeCodex, "OpenAI Codex"},
		{AssistantTypeUnknown, ""},
		{AssistantType("custom_type"), "custom_type"},
	}

	for _, tt := range tests {
		got := tt.atype.DisplayName()
		if got != tt.want {
			t.Errorf("AssistantType(%q).DisplayName() = %q, want %q", tt.atype, got, tt.want)
		}
	}
}

func TestAssistantType_String(t *testing.T) {
	if AssistantTypeClaudeCode.String() != "claude_code" {
		t.Errorf("AssistantTypeClaudeCode.String() = %q, want %q", AssistantTypeClaudeCode.String(), "claude_code")
	}
	if AssistantTypeCodex.String() != "codex" {
		t.Errorf("AssistantTypeCodex.String() = %q, want %q", AssistantTypeCodex.String(), "codex")
	}
}

func TestDetectAssistant_ClaudeCodeCLIJS(t *testing.T) {
	got, ok := DetectAssistant("claude-code/cli.js")
	if !ok {
		t.Error("expected detection of claude-code/cli.js")
	}
	if got != AssistantTypeClaudeCode {
		t.Errorf("got %q, want %q", got, AssistantTypeClaudeCode)
	}
}

func TestDetectAssistant_CodexJS(t *testing.T) {
	got, ok := DetectAssistant("codex.js")
	if !ok {
		t.Error("expected detection of codex.js")
	}
	if got != AssistantTypeCodex {
		t.Errorf("got %q, want %q", got, AssistantTypeCodex)
	}
}

func TestDetectAssistant_SubstringAmbiguity(t *testing.T) {
	tests := []struct {
		name    string
		command string
		found   bool
	}{
		{
			name:    "noclause does not trigger claude",
			command: "echo 'this has noclause in it'",
			found:   false,
		},
		{
			name:    "codexy does not trigger codex",
			command: "python codexy.py",
			found:   false,
		},
		{
			name:    "claude with leading slash in path",
			command: "/usr/bin/claude -p prompt",
			found:   true,
		},
		{
			name:    "codex with leading slash in path",
			command: "/usr/bin/codex -q prompt",
			found:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := DetectAssistant(tt.command)
			if ok != tt.found {
				t.Errorf("DetectAssistant(%q) found = %v, want %v", tt.command, ok, tt.found)
			}
		})
	}
}

func TestDetectAssistant_CodexBinPath(t *testing.T) {
	got, ok := DetectAssistant("codex/bin/codex.js")
	if !ok {
		t.Error("expected detection of codex/bin/codex.js")
	}
	if got != AssistantTypeCodex {
		t.Errorf("got %q, want %q", got, AssistantTypeCodex)
	}
}

func TestDetectAssistant_QwenCode(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    AssistantType
		found   bool
	}{
		{
			name:    "qwen code binary",
			command: "/usr/bin/qwen code chat",
			want:    AssistantTypeQwenCode,
			found:   true,
		},
		{
			name:    "qwen code npm package",
			command: "node /home/user/.npm-global/lib/node_modules/@alicloud/qwen-code/cli.js",
			want:    AssistantTypeQwenCode,
			found:   true,
		},
		{
			name:    "qwen-code hyphenated binary",
			command: "/usr/local/bin/qwen-code",
			want:    AssistantTypeQwenCode,
			found:   true,
		},
		{
			name:    "qwen_code underscore binary",
			command: "/home/user/.local/bin/qwen_code",
			want:    AssistantTypeQwenCode,
			found:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DetectAssistant(tt.command)
			if ok != tt.found {
				t.Errorf("DetectAssistant(%q) found = %v, want %v", tt.command, ok, tt.found)
			}
			if ok && got != tt.want {
				t.Errorf("DetectAssistant(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

func TestDetectAssistant_Gemini(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    AssistantType
		found   bool
	}{
		{
			name:    "gemini-cli binary",
			command: "gemini-cli --model gemini-pro",
			want:    AssistantTypeGemini,
			found:   true,
		},
		{
			name:    "gemini npm package",
			command: "node /home/user/.npm-global/lib/node_modules/@google/gemini-cli/cli.js",
			want:    AssistantTypeGemini,
			found:   true,
		},
		{
			name:    "gemini binary path",
			command: "/usr/local/bin/gemini",
			want:    AssistantTypeGemini,
			found:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DetectAssistant(tt.command)
			if ok != tt.found {
				t.Errorf("DetectAssistant(%q) found = %v, want %v", tt.command, ok, tt.found)
			}
			if ok && got != tt.want {
				t.Errorf("DetectAssistant(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

func TestDetectAssistant_Cursor(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    AssistantType
		found   bool
	}{
		{
			name:    "cursor-agent binary",
			command: "/opt/cursor/bin/cursor-agent",
			want:    AssistantTypeCursor,
			found:   true,
		},
		{
			name:    "cursor binary path",
			command: "/usr/local/bin/cursor",
			want:    AssistantTypeCursor,
			found:   true,
		},
		{
			name:    "cursor config path",
			command: "node /home/user/.cursor/extensions/agent/dist/index.js",
			want:    AssistantTypeCursor,
			found:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DetectAssistant(tt.command)
			if ok != tt.found {
				t.Errorf("DetectAssistant(%q) found = %v, want %v", tt.command, ok, tt.found)
			}
			if ok && got != tt.want {
				t.Errorf("DetectAssistant(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

func TestAssistantType_DisplayNameNewTypes(t *testing.T) {
	tests := []struct {
		atype AssistantType
		want  string
	}{
		{AssistantTypeQwenCode, "Qwen Code"},
		{AssistantTypeGemini, "Gemini"},
		{AssistantTypeCursor, "Cursor"},
	}

	for _, tt := range tests {
		got := tt.atype.DisplayName()
		if got != tt.want {
			t.Errorf("AssistantType(%q).DisplayName() = %q, want %q", tt.atype, got, tt.want)
		}
	}
}
