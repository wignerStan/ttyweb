package server

import "testing"

func TestValidate_TLSClientAuthWithoutTLS(t *testing.T) {
	opts := &Options{
		EnableTLSClientAuth: true,
		EnableTLS:           false,
	}

	err := opts.Validate()
	if err == nil {
		t.Fatal("Expected error when TLS client auth is enabled without TLS, got nil")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	opts := &Options{
		Address: "0.0.0.0",
		Port:    "8080",
	}

	err := opts.Validate()
	if err != nil {
		t.Fatalf("Expected no error for valid config, got: %s", err)
	}
}
