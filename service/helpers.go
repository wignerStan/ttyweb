package service

import (
	"crypto/rand"
	"fmt"
)

// generateID creates a random 16-character hex string using crypto/rand.
func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateID: crypto/rand: %w", err)
	}
	return fmt.Sprintf("%x", b), nil
}
