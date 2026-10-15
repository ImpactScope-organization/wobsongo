package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

func newTestAuth() *auth.Auth {
	return auth.New("test-secret", 15*time.Minute, 72*time.Hour)
}

func TestAuthService_Register_CreatesUserAndReturnsTokens(t *testing.T) {
	var created *model.User
	userRepo := &mockrepo.UserRepoerMock{
		CreateFunc: func(_ context.Context, user *model.User) error {
			created = user
			return nil
		},
	}

	s := NewAuthService(userRepo, newTestAuth())
	user, tokens, err := s.Register(t.Context(), "Alice", "alice@example.com", "hunter22")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != model.RoleUser {
		t.Errorf("expected new registrations to default to RoleUser, got %s", user.Role)
	}
	if created == nil || created.Email != "alice@example.com" {
		t.Errorf("expected repo to receive the new user, got %+v", created)
	}
	if user.PasswordHash == "" || user.PasswordHash == "hunter22" {
		t.Error("expected password to be hashed before storing")
	}
	if tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Error("expected a non-empty token pair")
	}
}

func TestAuthService_Register_ConflictOnDuplicateEmail(t *testing.T) {
	userRepo := &mockrepo.UserRepoerMock{
		CreateFunc: func(context.Context, *model.User) error {
			return data.ErrConflict
		},
	}

	s := NewAuthService(userRepo, newTestAuth())
	_, _, err := s.Register(t.Context(), "Alice", "alice@example.com", "hunter22")
	if !errors.Is(err, data.ErrConflict) {
		t.Errorf("expected data.ErrConflict, got %v", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	hash, err := auth.HashPassword("hunter22", auth.WithMinCost())
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	userRepo := &mockrepo.UserRepoerMock{
		GetByEmailFunc: func(context.Context, string) (*model.User, error) {
			return &model.User{
				Email:        "alice@example.com",
				PasswordHash: hash,
				Role:         model.RoleAdmin,
			}, nil
		},
	}

	s := NewAuthService(userRepo, newTestAuth())
	tokens, err := s.Login(t.Context(), "alice@example.com", "hunter22")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Error("expected a non-empty access token")
	}
}

func TestAuthService_Login_WrongPasswordMasksAsIncorrectCredentials(t *testing.T) {
	hash, err := auth.HashPassword("hunter22", auth.WithMinCost())
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	userRepo := &mockrepo.UserRepoerMock{
		GetByEmailFunc: func(context.Context, string) (*model.User, error) {
			return &model.User{Email: "alice@example.com", PasswordHash: hash}, nil
		},
	}

	s := NewAuthService(userRepo, newTestAuth())
	_, err = s.Login(t.Context(), "alice@example.com", "wrong-password")
	if !errors.Is(err, auth.ErrIncorrectCredentials) {
		t.Errorf("expected ErrIncorrectCredentials, got %v", err)
	}
}

func TestAuthService_Login_UnknownEmailMasksAsIncorrectCredentials(t *testing.T) {
	userRepo := &mockrepo.UserRepoerMock{
		GetByEmailFunc: func(context.Context, string) (*model.User, error) {
			return nil, data.ErrNotFound
		},
	}

	s := NewAuthService(userRepo, newTestAuth())
	_, err := s.Login(t.Context(), "nobody@example.com", "hunter22")
	if !errors.Is(err, auth.ErrIncorrectCredentials) {
		t.Errorf(
			"expected not-found to be masked as ErrIncorrectCredentials (no email enumeration), got %v",
			err,
		)
	}
}

func TestAuthService_Login_NormalizesEmail(t *testing.T) {
	var looked string
	userRepo := &mockrepo.UserRepoerMock{
		GetByEmailFunc: func(_ context.Context, email string) (*model.User, error) {
			looked = email
			return nil, data.ErrNotFound
		},
	}
	s := NewAuthService(userRepo, newTestAuth())
	_, _ = s.Login(t.Context(), " Alice@Example.com ", "hunter22")
	if looked != "alice@example.com" {
		t.Errorf("expected normalized lookup, got %q", looked)
	}
}

func TestAuthService_Register_NormalizesEmail(t *testing.T) {
	userRepo := &mockrepo.UserRepoerMock{
		CreateFunc: func(context.Context, *model.User) error { return nil },
	}
	s := NewAuthService(userRepo, newTestAuth())
	user, _, err := s.Register(t.Context(), "Alice", "ALICE@example.com", "hunter22-hunter22")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected normalized email, got %q", user.Email)
	}
}

// refreshFixture signs a token pair at issuedAt for a user whose password
// last changed at changedAt.
func refreshFixture(
	t *testing.T,
	issuedAt, changedAt time.Time,
) (*AuthService, *auth.JWTTokens, *model.User) {
	t.Helper()
	user := &model.User{
		ID:                uuid.New(),
		Name:              "Alice",
		Email:             "alice@example.com",
		Role:              model.RoleAdmin,
		PasswordChangedAt: changedAt,
	}
	a := newTestAuth().WithClock(func() time.Time { return issuedAt })
	tokens, err := a.GenerateJWTPair(&auth.JWTPayload{ID: user.ID, Role: model.RoleUser})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	userRepo := &mockrepo.UserRepoerMock{
		GetByIDFunc: func(_ context.Context, id uuid.UUID) (*model.User, error) {
			if id != user.ID {
				return nil, data.ErrNotFound
			}
			return user, nil
		},
	}
	return NewAuthService(userRepo, newTestAuth()), tokens, user
}

func TestAuthService_Refresh_IssuesTokensWithCurrentRole(t *testing.T) {
	now := time.Now()
	s, tokens, _ := refreshFixture(t, now, now.Add(-time.Hour))

	fresh, err := s.Refresh(t.Context(), tokens.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	claims, err := newTestAuth().ValidateJWT(fresh.AccessToken, string(auth.AccessTokenSubject))
	if err != nil {
		t.Fatalf("new access token invalid: %v", err)
	}
	if claims.Role != model.RoleAdmin {
		t.Errorf("expected the role from the database, got %s", claims.Role)
	}
}

func TestAuthService_Refresh_RejectsAccessTokens(t *testing.T) {
	now := time.Now()
	s, tokens, _ := refreshFixture(t, now, now.Add(-time.Hour))
	if _, err := s.Refresh(
		t.Context(),
		tokens.AccessToken,
	); !errors.Is(
		err,
		auth.ErrWrongTokenType,
	) {
		t.Errorf("expected ErrWrongTokenType, got %v", err)
	}
}

func TestAuthService_Refresh_RejectsTokensFromBeforePasswordChange(t *testing.T) {
	now := time.Now()
	s, tokens, _ := refreshFixture(t, now.Add(-time.Hour), now.Add(-time.Minute))
	if _, err := s.Refresh(
		t.Context(),
		tokens.RefreshToken,
	); !errors.Is(
		err,
		auth.ErrInvalidToken,
	) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_Refresh_KeepsTokensFromTheSameSecondAsTheChange(t *testing.T) {
	changed := time.Now().Truncate(time.Second).Add(700 * time.Millisecond)
	s, tokens, _ := refreshFixture(t, changed.Add(100*time.Millisecond), changed)
	if _, err := s.Refresh(t.Context(), tokens.RefreshToken); err != nil {
		t.Errorf("expected the token to be kept, got %v", err)
	}
}

func TestAuthService_Refresh_RejectsTokensOfDeletedUsers(t *testing.T) {
	now := time.Now()
	s, _, _ := refreshFixture(t, now, now.Add(-time.Hour))
	other, err := newTestAuth().GenerateJWTPair(&auth.JWTPayload{ID: uuid.New()})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	if _, err := s.Refresh(t.Context(), other.RefreshToken); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	hash, err := auth.HashPassword("old-password-1", auth.WithMinCost())
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}
	user := &model.User{ID: uuid.New(), PasswordHash: hash}
	var stored string
	userRepo := &mockrepo.UserRepoerMock{
		GetByIDFunc: func(context.Context, uuid.UUID) (*model.User, error) { return user, nil },
		UpdatePasswordFunc: func(_ context.Context, _ uuid.UUID, h string) (*model.User, error) {
			stored = h
			return &model.User{ID: user.ID, PasswordHash: h, PasswordChangedAt: time.Now()}, nil
		},
	}
	s := NewAuthService(userRepo, newTestAuth())

	if _, err := s.ChangePassword(t.Context(), user.ID, "wrong", "new-password-12"); !errors.Is(
		err, auth.ErrIncorrectCredentials,
	) {
		t.Errorf("wrong current password: expected ErrIncorrectCredentials, got %v", err)
	}
	if stored != "" {
		t.Error("must not store a password when the current one is wrong")
	}

	tokens, err := s.ChangePassword(t.Context(), user.ID, "old-password-1", "new-password-12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth.ComparePasswords("new-password-12", stored) != nil {
		t.Error("expected the new password to be hashed and stored")
	}
	if tokens == nil || tokens.RefreshToken == "" {
		t.Error("expected a fresh token pair")
	}
}

func TestAuthService_UpdateProfile_NormalizesEmail(t *testing.T) {
	var got string
	userRepo := &mockrepo.UserRepoerMock{
		UpdateProfileFunc: func(_ context.Context, _ uuid.UUID, _, email string) (*model.User, error) {
			got = email
			return &model.User{Email: email}, nil
		},
	}
	s := NewAuthService(userRepo, newTestAuth())
	if _, err := s.UpdateProfile(t.Context(), uuid.New(), "Alice", " Alice@X.org"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "alice@x.org" {
		t.Errorf("got %q", got)
	}
}
