package utils

import (
	"testing"
	"time"
)

func TestGenerateTokenExpiration(t *testing.T) {
	SecretKey = []byte("test-secret-key-that-is-at-least-32-chars")
	now := time.Now()

	shortToken, err := GenerateToken(1, "short", false)
	if err != nil {
		t.Fatal(err)
	}
	shortClaims, err := ValidateToken(shortToken)
	if err != nil {
		t.Fatal(err)
	}
	shortLifetime := shortClaims.ExpiresAt.Time.Sub(now)
	if shortLifetime < 23*time.Hour || shortLifetime > 25*time.Hour {
		t.Fatalf("short token lifetime = %s, want about 24h", shortLifetime)
	}

	longToken, err := GenerateToken(1, "long", true)
	if err != nil {
		t.Fatal(err)
	}
	longClaims, err := ValidateToken(longToken)
	if err != nil {
		t.Fatal(err)
	}
	longLifetime := longClaims.ExpiresAt.Time.Sub(now)
	if longLifetime < 9*365*24*time.Hour || longLifetime > 10*365*24*time.Hour+25*time.Hour {
		t.Fatalf("long token lifetime = %s, want about 10y", longLifetime)
	}
}
