package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

func TestGenerateJWTPair_ExpiriesFollowTTLs(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	a := New("secret", 15*time.Minute, 72*time.Hour).WithClock(func() time.Time { return now })

	tokens, err := a.GenerateJWTPair(&JWTPayload{ID: uuid.New(), Role: model.RoleAdmin})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tokens.AccessExpiresAt.Equal(now.Add(15 * time.Minute)) {
		t.Errorf("access expiry = %v", tokens.AccessExpiresAt)
	}
	if !tokens.RefreshExpiresAt.Equal(now.Add(72 * time.Hour)) {
		t.Errorf("refresh expiry = %v", tokens.RefreshExpiresAt)
	}

	claims, err := a.ValidateJWT(tokens.RefreshToken, string(RefreshTokenSubject))
	if err != nil {
		t.Fatalf("refresh token should validate: %v", err)
	}
	if claims.Role != model.RoleAdmin || !claims.ExpiresAt.Equal(tokens.RefreshExpiresAt) {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestValidateJWT_RejectsExpiredTokens(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	a := New("secret", 15*time.Minute, 72*time.Hour).WithClock(func() time.Time { return now })
	tokens, err := a.GenerateJWTPair(&JWTPayload{ID: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	later := a.WithClock(func() time.Time { return now.Add(16 * time.Minute) })
	if _, err := later.ValidateJWT(tokens.AccessToken); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("access token after 16 min: expected ErrTokenExpired, got %v", err)
	}
	if _, err := later.ValidateJWT(tokens.RefreshToken); err != nil {
		t.Errorf("refresh token after 16 min should still be valid: %v", err)
	}

	muchLater := a.WithClock(func() time.Time { return now.Add(73 * time.Hour) })
	if _, err := muchLater.ValidateJWT(tokens.RefreshToken); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("refresh token after 73 h: expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateJWT_RejectsWrongSubjectAndSecret(t *testing.T) {
	a := New("secret", 15*time.Minute, 72*time.Hour)
	tokens, err := a.GenerateJWTPair(&JWTPayload{ID: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := a.ValidateJWT(tokens.RefreshToken, string(AccessTokenSubject)); !errors.Is(
		err, ErrWrongTokenType,
	) {
		t.Errorf("refresh token used as access: expected ErrWrongTokenType, got %v", err)
	}
	other := New("other-secret", 15*time.Minute, 72*time.Hour)
	if _, err := other.ValidateJWT(tokens.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("foreign signature: expected ErrInvalidToken, got %v", err)
	}
}

func TestPasswordValid(t *testing.T) {
	cases := map[string]bool{
		"short":                  false,
		"twelve-chars":           true,
		"ééééééééééée":           true, // 12 characters, 24 bytes
		string(make([]byte, 73)): false,
		"exactly-72-bytes-" + string(make([]byte, 55)): true,
	}
	for password, want := range cases {
		if got := PasswordValid(password); got != want {
			t.Errorf("PasswordValid(%q) = %v, want %v", password, got, want)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Errorf("got %q", got)
	}
}
