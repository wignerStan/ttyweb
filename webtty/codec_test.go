package webtty

import (
	"bytes"
	"testing"
)

func TestNullCodecRoundTrip(t *testing.T) {
	t.Parallel()
	codec := NullCodec{}

	tests := []struct {
		name string
		src  []byte
	}{
		{"empty", []byte{}},
		{"ascii text", []byte("hello, world!")},
		{"binary zeros", bytes.Repeat([]byte{0x00}, 100)},
		{"binary 0xFF", bytes.Repeat([]byte{0xFF}, 100)},
		{"mixed binary", []byte{0x00, 0x01, 0x7F, 0x80, 0xFE, 0xFF}},
		{"null bytes in text", []byte("hello\x00world")},
		{"utf-8", []byte("\xc3\xa9\xc3\xa0\xc3\xbc")},
		{"large payload", bytes.Repeat([]byte("abcdefgh"), 256)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, len(tt.src)+100)

			// Encode.
			nEnc, err := codec.Encode(buf, tt.src)
			if err != nil {
				t.Fatalf("Encode() error: %v", err)
			}

			// Decode.
			decoded := make([]byte, len(tt.src)+100)
			nDec, err := codec.Decode(decoded, buf[:nEnc])
			if err != nil {
				t.Fatalf("Decode() error: %v", err)
			}

			if !bytes.Equal(decoded[:nDec], tt.src) {
				t.Errorf("round-trip mismatch:\n  src:  %x\n  got: %x", tt.src, decoded[:nDec])
			}
		})
	}
}

func TestNullCodecEncodeDstSmallerThanSrc(t *testing.T) {
	t.Parallel()
	codec := NullCodec{}
	src := []byte("hello, world!") // 13 bytes
	dst := make([]byte, 5)         // only 5 bytes

	n, err := codec.Encode(dst, src)
	if err != nil {
		t.Fatalf("Encode() unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("Encode() n = %d, want 5 (truncated)", n)
	}
	if !bytes.Equal(dst[:n], src[:5]) {
		t.Errorf("Encode() output = %q, want %q", dst[:n], src[:5])
	}
}

func TestNullCodecDecodeDstSmallerThanSrc(t *testing.T) {
	t.Parallel()
	codec := NullCodec{}
	src := []byte("decode me now") // 14 bytes
	dst := make([]byte, 3)         // only 3 bytes

	n, err := codec.Decode(dst, src)
	if err != nil {
		t.Fatalf("Decode() unexpected error: %v", err)
	}
	if n != 3 {
		t.Errorf("Decode() n = %d, want 3 (truncated)", n)
	}
}

func TestNullCodecEncodeSpecialChars(t *testing.T) {
	t.Parallel()
	codec := NullCodec{}

	special := []byte{
		'\n', '\r', '\t', '\x1b', // escape sequence
		0x7F,             // DEL
		0x00, 0x01, 0x02, // control chars
	}
	buf := make([]byte, len(special)+10)

	n, err := codec.Encode(buf, special)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}
	if !bytes.Equal(buf[:n], special) {
		t.Errorf("Encode() output = %x, want %x", buf[:n], special)
	}
}

func TestNullCodecMultipleSequentialOps(t *testing.T) {
	t.Parallel()
	codec := NullCodec{}

	messages := [][]byte{
		[]byte("first"),
		[]byte("second message"),
		[]byte("third"),
		{},
		[]byte("last"),
	}

	for _, msg := range messages {
		buf := make([]byte, len(msg)+10)
		n, err := codec.Encode(buf, msg)
		if err != nil {
			t.Fatalf("Encode() error for %q: %v", msg, err)
		}
		if n != len(msg) {
			t.Errorf("Encode() length = %d, want %d", n, len(msg))
		}

		decoded := make([]byte, len(msg)+10)
		nDec, err := codec.Decode(decoded, buf[:n])
		if err != nil {
			t.Fatalf("Decode() error for %q: %v", msg, err)
		}
		if !bytes.Equal(decoded[:nDec], msg) {
			t.Errorf("round-trip mismatch for %q", msg)
		}
	}
}

func TestNullCodecPreservesANSIEscapeSequences(t *testing.T) {
	t.Parallel()
	codec := NullCodec{}

	ansi := []byte("\x1b[31mred text\x1b[0m")
	buf := make([]byte, len(ansi)+10)

	n, err := codec.Encode(buf, ansi)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}
	if !bytes.Equal(buf[:n], ansi) {
		t.Errorf("ANSI escape sequences not preserved: got %q, want %q", buf[:n], ansi)
	}
}

func TestNullCodecLargeRandomContent(t *testing.T) {
	t.Parallel()
	codec := NullCodec{}

	// Build a large payload with all byte values.
	var src []byte
	for b := 0; b < 256; b++ {
		src = append(src, byte(b))
	}
	src = append(src, bytes.Repeat(src, 100)...)

	buf := make([]byte, len(src))
	n, err := codec.Encode(buf, src)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}
	if !bytes.Equal(buf[:n], src) {
		t.Error("large content encode mismatch")
	}

	decoded := make([]byte, len(src))
	nDec, err := codec.Decode(decoded, buf[:n])
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if !bytes.Equal(decoded[:nDec], src) {
		t.Error("large content round-trip mismatch")
	}
}

func TestNullCodecEmptyDestination(t *testing.T) {
	t.Parallel()
	codec := NullCodec{}
	var dst []byte

	n, err := codec.Encode(dst, []byte("x"))
	if err != nil {
		t.Fatalf("Encode() with nil dst unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("Encode() nil dst n = %d, want 0", n)
	}
}

func TestNullCodecEncodeSingleBytes(t *testing.T) {
	t.Parallel()
	codec := NullCodec{}

	for b := 0; b < 256; b++ {
		src := []byte{byte(b)}
		dst := make([]byte, 1)
		n, err := codec.Encode(dst, src)
		if err != nil {
			t.Fatalf("Encode(0x%02x) error: %v", b, err)
		}
		if n != 1 || dst[0] != byte(b) {
			t.Errorf("Encode(0x%02x) = %d/0x%02x, want 1/0x%02x", b, n, dst[0], b)
		}
	}
}

func TestNullCodecReusesBuffer(t *testing.T) {
	t.Parallel()
	codec := NullCodec{}
	buf := make([]byte, 100)

	// Encode first message.
	first := []byte("first message")
	n1, err := codec.Encode(buf, first)
	if err != nil {
		t.Fatalf("Encode() first error: %v", err)
	}

	// Encode second message into same buffer (shorter).
	second := []byte("second")
	n2, err := codec.Encode(buf, second)
	if err != nil {
		t.Fatalf("Encode() second error: %v", err)
	}

	if !bytes.Equal(buf[:n2], second) {
		t.Errorf("buffer reuse: got %q, want %q", buf[:n2], second)
	}

	// Byte just after n2 should still be leftover from the longer first message.
	if n1 > n2 && buf[n2] != first[n2] {
		t.Errorf("buffer reuse: buf[%d] = %q, want leftover %q", n2, buf[n2], first[n2])
	}
}
