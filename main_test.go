package main

import (
	"testing"
)

func TestHostname(t *testing.T) {
	h := hostname()
	if h == "" {
		t.Error("expected non-empty hostname")
	}
	if h == "localhost" {
		t.Log("hostname resolved to localhost (os.Hostname failed)")
	}
}

func TestDefaultShell_FromEnv(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	if sh := defaultShell(); sh != "/bin/zsh" {
		t.Errorf("expected /bin/zsh, got %s", sh)
	}
}

func TestDefaultShell_Fallback(t *testing.T) {
	t.Setenv("SHELL", "")
	if sh := defaultShell(); sh != "/bin/sh" {
		t.Errorf("expected /bin/sh fallback, got %s", sh)
	}
}

func TestCommandExists_True(t *testing.T) {
	if !commandExists("go") {
		t.Error("expected 'go' to exist in PATH")
	}
}

func TestCommandExists_False(t *testing.T) {
	if commandExists("nonexistent_binary_xyz123") {
		t.Error("expected nonexistent binary to not exist")
	}
}

func TestSelectBackend_Unknown(t *testing.T) {
	_, err := selectBackend("nonexistent", "", nil)
	if err == nil {
		t.Error("expected error for unknown backend")
	}
}

func TestSelectBackend_Local(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	b, err := selectBackend("local", "", []string{"bash"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil {
		t.Error("expected non-nil backend for local")
	}
}
