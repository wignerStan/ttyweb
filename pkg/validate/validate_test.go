package validate

import (
	"strings"
	"testing"
)

func TestSessionName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid cases
		{"alphanumeric", "mysession", false},
		{"with hyphens", "my-session", false},
		{"with dots", "my.session", false},
		{"with underscores", "my_session", false},
		{"with numbers", "session123", false},
		{"single char", "a", false},
		{"mixed", "Abc-123.z_9", false},

		// Invalid cases
		{"empty", "", true},
		{"spaces", "session with spaces", true},
		{"semicolons", "session;rm -rf /", true},
		{"path traversal", "../../etc/passwd", true},
		{"pipe", "foo|bar", true},
		{"ampersand", "foo&bar", true},
		{"backtick", "foo`bar`", true},
		{"dollar", "foo$bar", true},
		{"newline", "foo\nbar", true},
		{"tab", "foo\tbar", true},
		{"null byte", "foo\x00bar", true},
		{"slash", "foo/bar", true},
		{"backslash", "foo\\bar", true},
		{"at sign", "foo@bar", true},
		{"hash", "foo#bar", true},
		{"exclamation", "foo!bar", true},
		{"tilde", "foo~bar", true},
		{"parentheses", "foo(bar)", true},
		{"braces", "foo{bar}", true},
		{"brackets", "foo[bar]", true},
		{"angle brackets", "foo<bar>", true},
		{"too long", strings.Repeat("a", 129), true},
		{"exactly 128", strings.Repeat("a", 128), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SessionName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SessionName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestPaneID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid cases
		{"numeric", "0", false},
		{"with percent", "%0", false},
		{"window colon pane", "1:0", false},
		{"session colon pane", "session:0.1", false},
		{"complex", "my-session:0.1", false},
		{"with underscore", "my_pane", false},
		{"with dot", "pane.1", false},

		// Invalid cases
		{"empty", "", true},
		{"semicolon", "pane;rm", true},
		{"path traversal", "../../bad", true},
		{"spaces", "pane id", true},
		{"pipe", "pane|cmd", true},
		{"ampersand", "pane&cmd", true},
		{"backtick", "pane`cmd`", true},
		{"newline", "pane\nid", true},
		{"null byte", "pane\x00id", true},
		{"slash", "pane/id", true},
		{"dollar", "pane$var", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PaneID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("PaneID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
