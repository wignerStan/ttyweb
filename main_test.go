package main

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultShellWithEnv(t *testing.T) {
	t.Parallel()

	// Save and restore SHELL env var.
	origShell := os.Getenv("SHELL")
	defer func() {
		if origShell == "" {
			_ = os.Unsetenv("SHELL")
		} else {
			_ = os.Setenv("SHELL", origShell)
		}
	}()

	customShell := "/usr/local/bin/zsh"
	_ = os.Setenv("SHELL", customShell)

	got := defaultShell()
	if got != customShell {
		t.Errorf("defaultShell() = %q, want %q", got, customShell)
	}
}

func TestDefaultShellEmptyEnv(t *testing.T) {
	t.Parallel()

	origShell := os.Getenv("SHELL")
	defer func() {
		if origShell == "" {
			_ = os.Unsetenv("SHELL")
		} else {
			_ = os.Setenv("SHELL", origShell)
		}
	}()

	_ = os.Unsetenv("SHELL")

	got := defaultShell()
	if got != "/bin/sh" {
		t.Errorf("defaultShell() = %q, want %q", got, "/bin/sh")
	}
}

func TestHostnameNotEmpty(t *testing.T) {
	t.Parallel()
	got := hostname()
	if got == "" {
		t.Error("hostname() returned empty string")
	}
}

func TestHostnameIsString(t *testing.T) {
	t.Parallel()
	got := hostname()
	// Hostname should be a non-empty printable string.
	if len(got) == 0 || strings.TrimSpace(got) != got {
		t.Errorf("hostname() = %q, want a clean hostname string", got)
	}
}

func TestCommandExistsTrue(t *testing.T) {
	t.Parallel()
	// "sh" should always exist on any Unix system.
	if !commandExists("sh") {
		t.Error("commandExists(\"sh\") = false, want true")
	}
}

func TestCommandExistsFalse(t *testing.T) {
	t.Parallel()
	if commandExists("nonexistent_binary_that_does_not_exist_xyzzy") {
		t.Error("commandExists() returned true for a nonexistent binary")
	}
}

func TestCommandExistsEmpty(t *testing.T) {
	t.Parallel()
	if commandExists("") {
		t.Error("commandExists(\"\") should return false")
	}
}

func TestCommandExistsPathBinary(t *testing.T) {
	t.Parallel()
	// "ls" is a standard Unix utility.
	if !commandExists("ls") {
		t.Error("commandExists(\"ls\") = false, want true")
	}
}
