package webtty

import (
	"bytes"
	"testing"
)

func TestNullCodecEncode(t *testing.T) {
	codec := NullCodec{}

	tests := []struct {
		name string
		src  []byte
	}{
		{"empty input", []byte{}},
		{"single byte", []byte{0x42}},
		{"multi-byte", []byte("hello, world")},
		{"binary data", []byte{0x00, 0x01, 0xFF, 0xFE}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, len(tt.src)+10) // extra space to ensure no overflow
			n, err := codec.Encode(dst, tt.src)
			if err != nil {
				t.Fatalf("Unexpected error from Encode(): %s", err)
			}
			if n != len(tt.src) {
				t.Fatalf("Encode returned wrong length: expected %d, got %d", len(tt.src), n)
			}
			if !bytes.Equal(dst[:n], tt.src) {
				t.Fatalf("Encode output mismatch: expected %q, got %q", tt.src, dst[:n])
			}
		})
	}
}

func TestNullCodecDecode(t *testing.T) {
	codec := NullCodec{}

	tests := []struct {
		name string
		src  []byte
	}{
		{"empty input", []byte{}},
		{"single byte", []byte{0x42}},
		{"multi-byte", []byte("decode me")},
		{"binary data", []byte{0xCA, 0xFE, 0xBA, 0xBE}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, len(tt.src)+10)
			n, err := codec.Decode(dst, tt.src)
			if err != nil {
				t.Fatalf("Unexpected error from Decode(): %s", err)
			}
			if n != len(tt.src) {
				t.Fatalf("Decode returned wrong length: expected %d, got %d", len(tt.src), n)
			}
			if !bytes.Equal(dst[:n], tt.src) {
				t.Fatalf("Decode output mismatch: expected %q, got %q", tt.src, dst[:n])
			}
		})
	}
}
