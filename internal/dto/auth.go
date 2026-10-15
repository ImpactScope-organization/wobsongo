package dto

import (
	"time"

	"github.com/google/uuid"
)

// LoginRequest is the body of POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,max=255"`
	Password string `json:"password" validate:"required,max=72"`
}

// SignupRequest is the body of POST /api/v1/auth/signup. The password policy
// (auth.PasswordValid) is checked by the handler, so its message is specific.
type SignupRequest struct {
	Name     string `json:"name"     validate:"required,max=100"`
	Email    string `json:"email"    validate:"required,email,max=255"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest is the body of POST /api/v1/auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// UpdateProfileRequest is the body of PATCH /api/v1/me.
type UpdateProfileRequest struct {
	Name  string `json:"name"  validate:"required,max=100"`
	Email string `json:"email" validate:"required,email,max=255"`
}

// ChangePasswordRequest is the body of POST /api/v1/me/password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,max=72"`
	NewPassword     string `json:"new_password"     validate:"required"`
}

// TokenPairResponse is returned whenever tokens are issued. Both are JWTs;
// the access token goes in "Authorization: Bearer <token>", the refresh
// token only to POST /api/v1/auth/refresh.
type TokenPairResponse struct {
	AccessToken      string    `json:"access_token"       validate:"required"`
	AccessExpiresAt  time.Time `json:"access_expires_at"  validate:"required"`
	RefreshToken     string    `json:"refresh_token"      validate:"required"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at" validate:"required"`
}

// MeResponse is the signed-in user's own account.
type MeResponse struct {
	ID    uuid.UUID `json:"id"    validate:"required" format:"uuid"`
	Name  string    `json:"name"  validate:"required"`
	Email string    `json:"email" validate:"required"`
	// Role is one of "user", "admin" or "superadmin".
	Role              string    `json:"role"                validate:"required" enums:"user,admin,superadmin"`
	PasswordChangedAt time.Time `json:"password_changed_at" validate:"required"`
}
