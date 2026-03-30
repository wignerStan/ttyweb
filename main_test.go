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

func TestHostname_AlwaysString(t *testing.T) {
	t.Parallel()
	got := hostname()
	// hostname() returns os.Hostname() or "localhost" (the error path).
	// Any non-empty value that isn't "localhost" means os.Hostname() succeeded.
	if got == "" {
		t.Error("hostname() returned empty string")
	}
}

func TestHostnameIsString(t *testing.T) {
	t.Parallel()
	got := hostname()
	// Hostname should be a non-empty printable string.
	if got == "" || strings.TrimSpace(got) != got {
		t.Errorf("hostname() = %q, want a clean hostname string", got)
	}
}

func TestHostname_NotEmptyAndValid(t *testing.T) {
	t.Parallel()
	got := hostname()
	if got == "" {
		t.Fatal("hostname() returned empty string")
	}
	if len(got) > 253 {
		t.Errorf("hostname() = %q, exceeds max hostname length", got)
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

func TestSelectBackend_Local(t *testing.T) {
	t.Parallel()
	f, err := selectBackend("local", "s1", []string{"/bin/cat"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil factory")
	}
}

func TestSelectBackend_LocalDefaultShell(t *testing.T) {
	t.Parallel()
	f, err := selectBackend("local", "s1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil factory")
	}
}

func TestSelectBackend_Tmux(t *testing.T) {
	t.Parallel()
	f, err := selectBackend("tmux", "s1", nil)
	if !commandExists("tmux") {
		if err == nil {
			t.Error("expected error when tmux not found")
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil factory")
	}
}

func TestSelectBackend_Zellij(t *testing.T) {
	t.Parallel()
	f, err := selectBackend("zellij", "s1", nil)
	if !commandExists("zellij") {
		if err == nil {
			t.Error("expected error when zellij not found")
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil factory")
	}
}

func TestSelectBackend_Unknown(t *testing.T) {
	t.Parallel()
	_, err := selectBackend("unknown_backend", "s1", nil)
	if err == nil {
		t.Error("expected error for unknown backend")
	}
}

func TestSelectBackend_LocalEmptyArgsUsesDefaultShell(t *testing.T) {
	t.Parallel()
	f, err := selectBackend("local", "s1", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil factory")
	}
}

func TestBuildOptions_Basic(t *testing.T) {
	t.Parallel()
	opts := buildOptions("0.0.0.0", "8080", "/", "local", "", "title", false, false, "", "")
	if opts.Address != "0.0.0.0" {
		t.Errorf("Address = %q, want '0.0.0.0'", opts.Address)
	}
	if opts.Port != "8080" {
		t.Errorf("Port = %q, want '8080'", opts.Port)
	}
	if !opts.PermitArguments {
		t.Error("expected PermitArguments true for local backend")
	}
	if opts.EnableBasicAuth {
		t.Error("expected EnableBasicAuth false")
	}
}

func TestBuildOptions_WithCredential(t *testing.T) {
	t.Parallel()
	opts := buildOptions("", "", "", "local", "user:pass", "", false, false, "", "")
	if !opts.EnableBasicAuth {
		t.Error("expected EnableBasicAuth true")
	}
	if opts.Credential != "user:pass" {
		t.Errorf("Credential = %q, want 'user:pass'", opts.Credential)
	}
}

func TestBuildOptions_WithTLS(t *testing.T) {
	t.Parallel()
	opts := buildOptions("", "", "", "local", "", "", false, true, "/cert.pem", "/key.pem")
	if !opts.EnableTLS {
		t.Error("expected EnableTLS true")
	}
	if opts.TLSCrtFile != "/cert.pem" {
		t.Errorf("TLSCrtFile = %q, want '/cert.pem'", opts.TLSCrtFile)
	}
	if opts.TLSKeyFile != "/key.pem" {
		t.Errorf("TLSKeyFile = %q, want '/key.pem'", opts.TLSKeyFile)
	}
}

func TestBuildOptions_PermitWrite(t *testing.T) {
	t.Parallel()
	opts := buildOptions("", "", "", "local", "", "", true, false, "", "")
	if !opts.PermitWrite {
		t.Error("expected PermitWrite true")
	}
}

func TestBuildOptions_NonLocalBackend(t *testing.T) {
	t.Parallel()
	opts := buildOptions("", "", "", "tmux", "", "", false, false, "", "")
	if opts.PermitArguments {
		t.Error("expected PermitArguments false for tmux backend")
	}
}

func TestResolveDBPath_Empty(t *testing.T) {
	t.Parallel()
	result := resolveDBPath("")
	if result == "" {
		t.Error("expected non-empty default path")
	}
}

func TestResolveDBPath_Explicit(t *testing.T) {
	t.Parallel()
	result := resolveDBPath("/custom/path.db")
	if result != "/custom/path.db" {
		t.Errorf("expected '/custom/path.db', got %q", result)
	}
}

func TestBuildOptions_WithTLSNoCertKey(t *testing.T) {
	t.Parallel()
	opts := buildOptions("", "", "", "local", "", "", false, true, "", "")
	if !opts.EnableTLS {
		t.Error("expected EnableTLS true")
	}
	if opts.TLSCrtFile != "" {
		t.Errorf("TLSCrtFile = %q, want empty", opts.TLSCrtFile)
	}
	if opts.TLSKeyFile != "" {
		t.Errorf("TLSKeyFile = %q, want empty", opts.TLSKeyFile)
	}
}

func TestBuildOptions_WithTitleVariables(t *testing.T) {
	t.Parallel()
	opts := buildOptions("0.0.0.0", "8080", "/", "local", "", "test {{ .hostname }}", false, false, "", "")
	if opts.TitleFormat != "test {{ .hostname }}" {
		t.Errorf("TitleFormat = %q", opts.TitleFormat)
	}
	if _, ok := opts.TitleVariables["hostname"]; !ok {
		t.Error("expected hostname in TitleVariables")
	}
}

func TestLoadConfig_Default(t *testing.T) {
	t.Parallel()
	err := loadConfig("")
	if err != nil {
		t.Logf("loadConfig with default path: %v (may be expected)", err)
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfgPath := tmpDir + "/config.json"
	if err := os.WriteFile(cfgPath, []byte(`{"llm":{"api_url":"https://example.com","model":"test"}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := loadConfig(cfgPath); err != nil {
		t.Fatalf("loadConfig valid file: %v", err)
	}
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfgPath := tmpDir + "/config.json"
	if err := os.WriteFile(cfgPath, []byte(`{invalid json`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := loadConfig(cfgPath); err == nil {
		t.Error("expected error for invalid JSON config")
	}
}

func TestBuildOptions_EnvCredential(t *testing.T) {
	t.Parallel()
	orig := os.Getenv("TTYWEB_CREDENTIAL")
	defer os.Setenv("TTYWEB_CREDENTIAL", orig)
	os.Setenv("TTYWEB_CREDENTIAL", "env:user:pass")

	opts := buildOptions("", "", "", "local", "flag:user:pass", "", false, false, "", "")
	if !opts.EnableBasicAuth {
		t.Error("expected EnableBasicAuth true")
	}
	if opts.Credential != "env:user:pass" {
		t.Errorf("Credential = %q, want env var to take precedence", opts.Credential)
	}
}

func TestBuildOptions_DeprecatedCredentialFlag(t *testing.T) {
	t.Parallel()
	orig := os.Getenv("TTYWEB_CREDENTIAL")
	defer os.Setenv("TTYWEB_CREDENTIAL", orig)
	os.Unsetenv("TTYWEB_CREDENTIAL")

	opts := buildOptions("", "", "", "local", "user:pass", "", false, false, "", "")
	if !opts.EnableBasicAuth {
		t.Error("expected EnableBasicAuth true")
	}
	if opts.Credential != "user:pass" {
		t.Errorf("Credential = %q, want 'user:pass'", opts.Credential)
	}
}

func TestBuildOptions_TLSWithCertAndKey(t *testing.T) {
	t.Parallel()
	opts := buildOptions("", "", "", "local", "", "", false, true, "/crt", "/key")
	if !opts.EnableTLS {
		t.Error("expected EnableTLS true")
	}
	if opts.TLSCrtFile != "/crt" {
		t.Errorf("TLSCrtFile = %q", opts.TLSCrtFile)
	}
	if opts.TLSKeyFile != "/key" {
		t.Errorf("TLSKeyFile = %q", opts.TLSKeyFile)
	}
}
