package homedir

import (
	"os"
	"testing"
)

func TestExpand(t *testing.T) {
	home := os.Getenv("HOME")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tilde slash expands to home",
			input: "~/foo",
			want:  home + "/foo",
		},
		{
			name:  "absolute path unchanged",
			input: "/tmp/foo",
			want:  "/tmp/foo",
		},
		{
			name:  "relative path unchanged",
			input: "foo/bar",
			want:  "foo/bar",
		},
		{
			name:  "empty string does not panic",
			input: "",
			want:  "",
		},
		{
			name:  "single character does not panic",
			input: "x",
			want:  "x",
		},
		{
			name:  "bare tilde does not expand",
			input: "~",
			want:  "~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Expand(tt.input)
			if got != tt.want {
				t.Errorf("Expand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
