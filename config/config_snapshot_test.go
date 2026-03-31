package config

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGoldenCfg = flag.Bool("update", false, "update golden files")

func compareGoldenCfg(t *testing.T, got []byte) {
	t.Helper()
	golden := filepath.Join("testdata", t.Name()+".golden")
	if *updateGoldenCfg {
		t.Logf("updating golden file: %s", golden)
		_ = os.MkdirAll(filepath.Dir(golden), 0o755)
		_ = os.WriteFile(golden, got, 0o644)
	}
	want, err := os.ReadFile(golden) //nolint:gosec // reason: test code, golden file path is deterministic
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestSnapshot_DefaultConfigPath(t *testing.T) {
	// Use a fixed HOME so the golden file is portable across environments.
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("XDG_CONFIG_HOME", "")

	path := DefaultConfigPath()

	compareGoldenCfg(t, []byte(path))
}

func TestSnapshot_DefaultConfigPath_WithXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-snapshot-test")

	path := DefaultConfigPath()

	compareGoldenCfg(t, []byte(path))
}

func TestSnapshot_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	compareGoldenCfg(t, data)
}
