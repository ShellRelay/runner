package main

import (
	"strings"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	t.Run("starts with sr_ prefix", func(t *testing.T) {
		tok, err := generateToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(tok, "sr_") {
			t.Errorf("token %q does not start with sr_", tok)
		}
	})

	t.Run("has correct length (sr_ + 46 hex chars = 49)", func(t *testing.T) {
		tok, err := generateToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// "sr_" (3) + 23 bytes hex encoded (46) = 49
		if len(tok) != 49 {
			t.Errorf("token length = %d, want 49 (got %q)", len(tok), tok)
		}
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			tok, err := generateToken()
			if err != nil {
				t.Fatalf("unexpected error on iteration %d: %v", i, err)
			}
			if seen[tok] {
				t.Fatalf("duplicate token generated: %q", tok)
			}
			seen[tok] = true
		}
	})

	t.Run("hex portion contains only valid hex chars", func(t *testing.T) {
		tok, err := generateToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		hex := tok[3:] // strip "sr_"
		for _, c := range hex {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("invalid hex char %c in token hex portion %q", c, hex)
			}
		}
	})

	t.Run("meets minimum length requirement for announce endpoint", func(t *testing.T) {
		tok, err := generateToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Server requires token.startsWith('sr_') && token.length >= 20
		if len(tok) < 20 {
			t.Errorf("token length %d < 20 minimum", len(tok))
		}
	})
}
