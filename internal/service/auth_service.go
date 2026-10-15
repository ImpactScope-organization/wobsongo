package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// AuthService handles registration, sign-in and the signed-in user's own
// account, for both the HTML pages and the JSON API.
type AuthService struct {
	userRepo data.UserRepoer
	auth     *auth.Auth
}

// NewAuthService creates an AuthService.
func NewAuthService(userRepo data.UserRepoer, a *auth.Auth) *AuthService {
	return &AuthService{userRepo: userRepo, auth: a}
}

// Register creates a new user account with the default role and returns a
// JWT token pair. Returns data.ErrConflict if the email is already registered.
func (s *AuthService) Register(
	ctx context.Context,
	name, email, password string,
) (*model.User, *auth.JWTTokens, error) {
	user, err := s.createUser(ctx, name, email, password, model.RoleUser)
	if err != nil {
		return nil, nil, err
	}
	tokens, err := s.issueTokens(user)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

// CreateSuperadmin creates a new superadmin account — used only by the
// createsuperadmin CLI command to bootstrap the first operator account, never
// through self-registration. Returns data.ErrConflict if the email is already
// registered.
func (s *AuthService) CreateSuperadmin(
	ctx context.Context,
	name, email, password string,
) (*model.User, error) {
	return s.createUser(ctx, name, email, password, model.RoleSuperadmin)
}

func (s *AuthService) createUser(
	ctx context.Context,
	name, email, password string,
	role model.UserRole,
) (*model.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", data.ErrInternal, err)
	}
	user := &model.User{
		Name:         name,
		Email:        auth.NormalizeEmail(email),
		Role:         role,
		PasswordHash: hash,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err // already mapped by repo layer (ErrConflict, ErrInternal)
	}
	return user, nil
}

// Login validates credentials and returns a JWT token pair.
func (s *AuthService) Login(ctx context.Context, email, password string) (*auth.JWTTokens, error) {
	user, err := s.userRepo.GetByEmail(ctx, auth.NormalizeEmail(email))
	if err != nil {
		// Mask not-found as incorrect credentials to prevent email enumeration.
		return nil, auth.ErrIncorrectCredentials
	}
	if err := auth.ComparePasswords(password, user.PasswordHash); err != nil {
		return nil, auth.ErrIncorrectCredentials
	}
	return s.issueTokens(user)
}

// Refresh exchanges a refresh token for a new token pair. The user is read
// again, so the new tokens carry their current name and role. Returns
// auth.ErrInvalidToken if the user is gone or changed their password after
// the token was issued, and auth.ErrTokenExpired once it has expired.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*auth.JWTTokens, error) {
	claims, err := s.auth.ValidateJWT(refreshToken, string(auth.RefreshTokenSubject))
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if errors.Is(err, data.ErrNotFound) {
		return nil, auth.ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	// JWT times have second precision, so compare at that precision: a token
	// issued in the same second as the change (e.g. by ChangePassword) is kept.
	if claims.IssuedAt.Before(user.PasswordChangedAt.Truncate(time.Second)) {
		return nil, auth.ErrInvalidToken
	}
	return s.issueTokens(user)
}

// Me returns the user with the given ID. Returns data.ErrNotFound if none exists.
func (s *AuthService) Me(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// UpdateProfile changes the user's name and email. Returns data.ErrConflict
// if another account uses the email.
func (s *AuthService) UpdateProfile(
	ctx context.Context,
	id uuid.UUID,
	name, email string,
) (*model.User, error) {
	return s.userRepo.UpdateProfile(ctx, id, name, auth.NormalizeEmail(email))
}

// ChangePassword sets a new password after checking the current one, and
// returns a fresh token pair: refresh tokens issued before the change,
// including the caller's, stop working. Returns auth.ErrIncorrectCredentials
// if the current password is wrong.
func (s *AuthService) ChangePassword(
	ctx context.Context,
	id uuid.UUID,
	currentPassword, newPassword string,
) (*auth.JWTTokens, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := auth.ComparePasswords(currentPassword, user.PasswordHash); err != nil {
		return nil, auth.ErrIncorrectCredentials
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", data.ErrInternal, err)
	}
	user, err = s.userRepo.UpdatePassword(ctx, id, hash)
	if err != nil {
		return nil, err
	}
	return s.issueTokens(user)
}

func (s *AuthService) issueTokens(user *model.User) (*auth.JWTTokens, error) {
	tokens, err := s.auth.GenerateJWTPair(&auth.JWTPayload{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Role:  user.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", data.ErrInternal, err)
	}
	return tokens, nil
}
