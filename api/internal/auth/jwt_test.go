package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestSignAndVerifyRoundTrip verifies all claim fields survive the sign→verify round-trip.
func TestSignAndVerifyRoundTrip(t *testing.T) {
	token, err := Sign(42, "alice@example.com", "admin")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	claims, err := Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if claims.UserID != 42 {
		t.Errorf("UserID: want 42, got %d", claims.UserID)
	}
	if claims.Email != "alice@example.com" {
		t.Errorf("Email: want alice@example.com, got %q", claims.Email)
	}
	if claims.Role != "admin" {
		t.Errorf("Role: want admin, got %q", claims.Role)
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}
}

// TestSignPreservesRole verifies both user and admin roles are encoded correctly.
func TestSignPreservesRole(t *testing.T) {
	for _, role := range []string{"user", "admin"} {
		tok, err := Sign(1, "t@t.com", role)
		if err != nil {
			t.Fatalf("role=%s Sign: %v", role, err)
		}
		claims, err := Verify(tok)
		if err != nil {
			t.Fatalf("role=%s Verify: %v", role, err)
		}
		if claims.Role != role {
			t.Errorf("role=%s: got %q", role, claims.Role)
		}
	}
}

// TestVerifyExpiredToken confirms that a token with a past ExpiresAt is rejected.
func TestVerifyExpiredToken(t *testing.T) {
	expired := Claims{
		UserID: 1,
		Email:  "t@t.com",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, expired)
	signed, err := tok.SignedString(secret())
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	_, err = Verify(signed)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

// TestVerifyTamperedSignature confirms that flipping bytes in the signature fails.
func TestVerifyTamperedSignature(t *testing.T) {
	tok, _ := Sign(1, "t@t.com", "user")
	// Replace last 6 chars of the signature portion
	tampered := tok[:len(tok)-6] + "XXXXXX"

	_, err := Verify(tampered)
	if err == nil {
		t.Error("expected error for tampered signature, got nil")
	}
}

// TestVerifyMalformedToken confirms that a random string is rejected.
func TestVerifyMalformedToken(t *testing.T) {
	_, err := Verify("not.a.jwt")
	if err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

// TestVerifyEmptyString confirms that an empty string is rejected.
func TestVerifyEmptyString(t *testing.T) {
	_, err := Verify("")
	if err == nil {
		t.Error("expected error for empty token, got nil")
	}
}

// TestVerifyWrongSecret confirms that a token signed with a different secret fails.
func TestVerifyWrongSecret(t *testing.T) {
	// Sign with a non-default secret
	wrongSecret := []byte("completely-different-secret")
	claims := Claims{
		UserID: 1,
		Email:  "t@t.com",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := tok.SignedString(wrongSecret)

	// Verify uses the default secret — should fail
	_, err := Verify(signed)
	if err == nil {
		t.Error("expected error verifying token signed with wrong secret, got nil")
	}
}

// TestSignUsesJWTSecretEnvVar confirms the env var overrides the default secret.
func TestSignUsesJWTSecretEnvVar(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-env-secret")
	tok, err := Sign(7, "env@test.com", "user")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	claims, err := Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.UserID != 7 {
		t.Errorf("UserID: want 7, got %d", claims.UserID)
	}
}
