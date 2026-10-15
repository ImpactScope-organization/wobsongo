package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

// BearerAuthMiddleware requires a valid access token in the
// "Authorization: Bearer <token>" header and makes its claims available via
// ClaimsFromContext. Anything else is a 401.
func BearerAuthMiddleware(a *auth.Auth) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			const prefix = "Bearer "
			header := c.Request().Header.Get(echo.HeaderAuthorization)
			if !strings.HasPrefix(header, prefix) {
				return &model.APIError{
					Code:     http.StatusUnauthorized,
					Internal: errors.New("missing bearer token"),
					Public:   "missing bearer token",
				}
			}
			claims, err := a.ValidateJWT(
				strings.TrimPrefix(header, prefix),
				string(auth.AccessTokenSubject),
			)
			if err != nil {
				return &model.APIError{
					Code:     http.StatusUnauthorized,
					Internal: err,
					Public:   "invalid or expired token",
				}
			}
			return next(&withAuthUser{Context: c, User: claims})
		}
	}
}

// RequireRoleMiddleware allows only users whose role is at least minimum.
// Use after BearerAuthMiddleware; without claims it answers 401.
func RequireRoleMiddleware(minimum model.UserRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := ClaimsFromContext(c)
			if claims == nil {
				return &model.APIError{
					Code:     http.StatusUnauthorized,
					Internal: errors.New("no authenticated user"),
					Public:   "missing bearer token",
				}
			}
			if !claims.Role.AtLeast(minimum) {
				return &model.APIError{
					Code: http.StatusForbidden,
					Internal: errors.New(
						"role " + claims.Role.String() + " below " + minimum.String(),
					),
					Public: "Forbidden: insufficient permissions",
				}
			}
			return next(c)
		}
	}
}

// authRateLimit limits the unauthenticated auth endpoints to 10 requests a
// minute per IP, burst 5, to slow down password guessing. No-op in testing.
func authRateLimit(cfg *internal.Config) echo.MiddlewareFunc {
	if cfg.IsTesting() {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}
	store := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(float64(10) / time.Minute.Seconds()),
		Burst:     5,
		ExpiresIn: time.Minute,
	})
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		IdentifierExtractor: func(c echo.Context) (string, error) {
			if ip := c.RealIP(); ip != "" {
				return ip, nil
			}
			return "unknown", nil
		},
		Store: store,
		DenyHandler: func(_ echo.Context, _ string, err error) error {
			return &model.APIError{
				Code:     http.StatusTooManyRequests,
				Internal: err,
				Public:   "too many attempts, try again in a minute",
			}
		},
	})
}
