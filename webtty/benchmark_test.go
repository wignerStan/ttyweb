package webtty

import (
	"encoding/base64"
	"testing"
)

func BenchmarkEncodeBase64(b *testing.B) {
	data := []byte("Hello, World! This is a terminal output benchmark test string with some longer content.")
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = base64.StdEncoding.EncodeToString(data)
	}
}

func BenchmarkDecodeBase64(b *testing.B) {
	data := []byte("Hello, World! This is a terminal output benchmark test string with some longer content.")
	encoded := base64.StdEncoding.EncodeToString(data)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = base64.StdEncoding.DecodeString(encoded)
	}
}
