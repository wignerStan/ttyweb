package server

import (
	"reflect"
	"testing"
)

func TestListAddresses(t *testing.T) {
	// listAddresses returns network interface addresses.
	// It should not panic and should return a non-nil slice.
	addrs := listAddresses()
	if addrs == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTitleVariables(t *testing.T) {
	srv := newTestServer()

	order := []string{"server", "client"}
	varUnits := map[string]map[string]any{
		"server": {"host": "localhost", "port": 8080},
		"client": {"ip": "127.0.0.1"},
	}

	result, err := srv.titleVariables(order, varUnits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["host"] != "localhost" {
		t.Fatalf("expected host=localhost, got %v", result["host"])
	}
	if result["port"] != 8080 {
		t.Fatalf("expected port=8080, got %v", result["port"])
	}
	if result["ip"] != "127.0.0.1" {
		t.Fatalf("expected ip=127.0.0.1, got %v", result["ip"])
	}

	// Later keys should override earlier ones.
	if !reflect.DeepEqual(result["server"], varUnits["server"]) {
		t.Fatal("expected server key to contain server varUnit")
	}
	if !reflect.DeepEqual(result["client"], varUnits["client"]) {
		t.Fatal("expected client key to contain client varUnit")
	}
}

func TestTitleVariables_Override(t *testing.T) {
	srv := newTestServer()

	order := []string{"a", "b"}
	varUnits := map[string]map[string]any{
		"a": {"key": "value-a"},
		"b": {"key": "value-b"},
	}

	result, err := srv.titleVariables(order, varUnits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 'b' comes after 'a', so 'b' should win.
	if result["key"] != "value-b" {
		t.Fatalf("expected key=value-b (later overrides), got %v", result["key"])
	}
}

func TestTitleVariables_MissingKey(t *testing.T) {
	srv := newTestServer()
	order := []string{"missing"}
	varUnits := map[string]map[string]any{
		"other": {},
	}
	_, err := srv.titleVariables(order, varUnits)
	if err == nil {
		t.Fatal("expected error for missing varUnit key, got nil")
	}
}
