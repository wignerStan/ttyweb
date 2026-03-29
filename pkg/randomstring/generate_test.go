package randomstring

import (
	"fmt"
	"regexp"
	"testing"
)

var hexCharRe = regexp.MustCompile(`^[0-9a-z]+$`)

func TestGenerateLength(t *testing.T) {
	t.Parallel()
	tests := []struct {
		length int
	}{
		{1},
		{8},
		{16},
		{32},
		{64},
		{128},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("len=%d", tc.length), func(t *testing.T) {
			result := Generate(tc.length)
			if len(result) != tc.length {
				t.Errorf("Generate(%d) returned %d characters, want %d", tc.length, len(result), tc.length)
			}
		})
	}
}

func TestGenerateZeroLength(t *testing.T) {
	t.Parallel()
	result := Generate(0)
	if result != "" {
		t.Errorf("Generate(0) = %q, want empty string", result)
	}
}

func TestGenerateContainsOnlyHexChars(t *testing.T) {
	t.Parallel()
	for i := 0; i < 50; i++ {
		result := Generate(32)
		if !hexCharRe.MatchString(result) {
			t.Errorf("Generate(32) produced invalid characters: %q", result)
		}
	}
}

func TestGenerateUniqueness(t *testing.T) {
	t.Parallel()
	const count = 100
	const length = 16
	seen := make(map[string]bool, count)

	for i := 0; i < count; i++ {
		result := Generate(length)
		if seen[result] {
			t.Errorf("Generate(%d) produced duplicate: %q (iteration %d)", length, result, i)
		}
		seen[result] = true
	}

	if len(seen) != count {
		t.Errorf("expected %d unique strings, got %d", count, len(seen))
	}
}

func TestGenerateDifferentLengthsUnique(t *testing.T) {
	t.Parallel()
	// Strings of different lengths should never collide.
	short := Generate(4)
	long := Generate(64)

	if len(short) == len(long) {
		t.Errorf("lengths should differ: short=%d, long=%d", len(short), len(long))
	}
	if short == long {
		t.Error("strings of different lengths should never be equal")
	}
}

func TestGenerateDistribution(t *testing.T) {
	t.Parallel()
	// With 36 possible characters and enough samples, we should see variety.
	const length = 100
	const iterations = 10
	allChars := make(map[rune]int)

	for i := 0; i < iterations; i++ {
		result := Generate(length)
		for _, ch := range result {
			allChars[ch]++
		}
	}

	// We should see at least 10 distinct characters out of 36 possible.
	if len(allChars) < 10 {
		t.Errorf("distribution too narrow: only %d distinct characters in %d samples", len(allChars), iterations*length)
	}
}

func TestGenerateNegativeLengthPanics(t *testing.T) {
	t.Parallel()
	// Negative length panics because make([]byte, negative) is invalid.
	// This documents the current behavior; if Generate is updated to handle
	// negatives gracefully, this test should be updated to expect an empty string.
	defer func() {
		if r := recover(); r == nil {
			t.Error("Generate(-1) should panic with negative length")
		}
	}()
	Generate(-1)
}
