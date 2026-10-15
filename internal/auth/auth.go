// Package auth provides JWT generation/validation for wobsongo's web layer.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

const tokenIssuer = "github.com/impactscope-organization/wobsongo"

// TokenSubject identifies the purpose of a JWT.
type TokenSubject string

const (
	AccessTokenSubject  TokenSubject = "wobsongo-access"
	RefreshTokenSubject TokenSubject = "wobsongo-refresh"
)

// JWTClaims are the claims embedded in every wobsongo web-layer JWT.
type JWTClaims struct {
	UserID uuid.UUID      `json:"user_id"`
	Name   string         `json:"name"`
	Email  string         `json:"email"`
	Role   model.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// JWTTokens holds an access/refresh token pair returned after login or registration.
type JWTTokens struct {
	AccessToken      string    `json:"access_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

// JWTPayload is the input for generating JWT tokens.
type JWTPayload struct {
	ID     uuid.UUID
	Name   string
	Email  string
	Role   model.UserRole
	AuthID uuid.UUID
}

// Auth holds the signing secret and token lifetimes used for all JWT operations.
//
// Both tokens are stateless: nothing is stored server-side, so a refresh
// token stays valid until it expires. Callers that need to cut sessions
// short (e.g. after a password change) compare IssuedAt against their own
// data.
type Auth struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

// New creates an Auth instance with the given access and refresh token lifetimes.
func New(secret string, accessTTL, refreshTTL time.Duration) *Auth {
	return &Auth{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}
}

// WithClock returns a copy of a that reads the time from now. For tests.
func (a *Auth) WithClock(now func() time.Time) *Auth {
	clone := *a
	clone.now = now
	return &clone
}

// GenerateJWTPair issues a new access + refresh token pair for the given payload.
func (a *Auth) GenerateJWTPair(payload *JWTPayload) (*JWTTokens, error) {
	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
	}
	payload.AuthID = id

	now := a.now()
	tokens := &JWTTokens{
		AccessExpiresAt:  now.Add(a.accessTTL),
		RefreshExpiresAt: now.Add(a.refreshTTL),
	}
	tokens.AccessToken, err = a.sign(
		a.claims(payload, AccessTokenSubject, now, tokens.AccessExpiresAt),
	)
	if err != nil {
		return nil, err
	}
	tokens.RefreshToken, err = a.sign(
		a.claims(payload, RefreshTokenSubject, now, tokens.RefreshExpiresAt),
	)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// ValidateJWT parses and validates a signed token. Optionally checks its subject.
func (a *Auth) ValidateJWT(signedToken string, purposes ...string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		new(JWTClaims),
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return a.secret, nil
		},
		jwt.WithTimeFunc(a.now),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Issuer != tokenIssuer {
		return nil, ErrInvalidToken
	}
	if len(purposes) > 0 && claims.Subject != purposes[0] {
		return nil, ErrWrongTokenType
	}
	return claims, nil
}

func (a *Auth) claims(
	payload *JWTPayload,
	subject TokenSubject,
	issuedAt, expiresAt time.Time,
) *JWTClaims {
	return &JWTClaims{
		UserID: payload.ID,
		Name:   payload.Name,
		Email:  payload.Email,
		Role:   payload.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   string(subject),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        payload.AuthID.String(),
		},
	}
}

func (a *Auth) sign(claims *JWTClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secret)
}
