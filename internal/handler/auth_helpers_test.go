package handler_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/labstack/echo/v4"
)

const testJWTSecret = "handler-test-secret"

// testAuth signs tokens the app under test accepts. Call useTestJWTSecret
// before building the app.
func testAuth() *auth.Auth {
	return auth.New(testJWTSecret, 15*time.Minute, 72*time.Hour)
}

func useTestJWTSecret(t *testing.T) {
	t.Helper()
	t.Setenv("APP_JWT_SECRET", testJWTSecret)
}

// bearerFor returns an "Authorization" header value for a user with role.
func bearerFor(t *testing.T, role model.UserRole) string {
	t.Helper()
	tokens, err := testAuth().GenerateJWTPair(&auth.JWTPayload{
		ID:    uuid.New(),
		Name:  "Test User",
		Email: "test@example.com",
		Role:  role,
	})
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return "Bearer " + tokens.AccessToken
}

// signInAs makes every request to e carry a bearer token for role, unless
// the request already sets Authorization. For tests of handlers whose auth
// isn't what's under test.
func signInAs(t *testing.T, e *echo.Echo, role model.UserRole) *echo.Echo {
	t.Helper()
	header := bearerFor(t, role)
	e.Pre(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Header.Get(echo.HeaderAuthorization) == "" {
				c.Request().Header.Set(echo.HeaderAuthorization, header)
			}
			return next(c)
		}
	})
	return e
}
