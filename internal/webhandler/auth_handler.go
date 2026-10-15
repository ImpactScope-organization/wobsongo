package webhandler

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"time"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/handler"
	"github.com/impactscope-organization/wobsongo/internal/service"
	authview "github.com/impactscope-organization/wobsongo/internal/view/auth"
	"github.com/labstack/echo/v4"
)

// AuthCookieName is the name of the HTTP cookie carrying the access token
// for HTML routes. Exported so core/app.go can reference it when wiring the
// cookie-based JWT middleware onto the web route group.
const AuthCookieName = "auth"

// AuthHandler handles HTML authentication routes.
type AuthHandler struct {
	config *internal.Config
	svc    *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *service.AuthService, cfg *internal.Config) *AuthHandler {
	return &AuthHandler{svc: svc, config: cfg}
}

// registerPage renders GET /register.
func (h *AuthHandler) registerPage(c echo.Context) error {
	if handler.ClaimsFromContext(c) != nil {
		return c.Redirect(http.StatusFound, "/")
	}
	return authview.Register(csrfToken(c), "").Render(c.Request().Context(), c.Response())
}

// registerPost handles POST /register — creates an account, sets the auth
// cookie, and redirects to / on success or re-renders the form on failure.
func (h *AuthHandler) registerPost(c echo.Context) error {
	ctx := c.Request().Context()
	name := c.FormValue("name")
	email := c.FormValue("email")
	password := c.FormValue("password")

	if name == "" {
		return authview.Register(csrfToken(c), "Name is required.").Render(ctx, c.Response())
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return authview.Register(csrfToken(c), "Please enter a valid email address.").
			Render(ctx, c.Response())
	}
	if !auth.PasswordValid(password) {
		return authview.Register(
			csrfToken(c),
			fmt.Sprintf("Password must be at least %d characters.", auth.PasswordMinLength),
		).Render(ctx, c.Response())
	}

	_, tokens, err := h.svc.Register(ctx, name, email, password)
	if err != nil {
		msg := msgSomethingWentWrong
		if errors.Is(err, data.ErrConflict) {
			msg = "An account with this email already exists."
		}
		return authview.Register(csrfToken(c), msg).Render(ctx, c.Response())
	}
	h.setAuthCookie(c, tokens)
	return c.Redirect(http.StatusFound, "/")
}

// loginPage renders GET /login.
func (h *AuthHandler) loginPage(c echo.Context) error {
	if handler.ClaimsFromContext(c) != nil {
		return c.Redirect(http.StatusFound, "/")
	}
	var errMsg string
	if c.QueryParam("error") != "" {
		errMsg = "Incorrect email or password. Try again."
	}
	return authview.Login(csrfToken(c), errMsg).Render(c.Request().Context(), c.Response())
}

// loginPost handles POST /login — validates credentials, sets the auth
// cookie, and redirects to / on success or back to /login?error=1 on failure.
func (h *AuthHandler) loginPost(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	tokens, err := h.svc.Login(c.Request().Context(), email, password)
	if err != nil {
		return c.Redirect(http.StatusFound, "/login?error=1")
	}

	h.setAuthCookie(c, tokens)
	return c.Redirect(http.StatusFound, "/")
}

// logoutPost handles POST /logout — clears the auth cookie and redirects to /login.
func (h *AuthHandler) logoutPost(c echo.Context) error {
	c.SetCookie(&http.Cookie{ //nolint:gosec // Secure omitted: app supports plain HTTP in dev
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   h.config.IsProduction(),
	})
	return c.Redirect(http.StatusFound, "/login")
}

func (h *AuthHandler) setAuthCookie(c echo.Context, tokens *auth.JWTTokens) {
	c.SetCookie(&http.Cookie{ //nolint:gosec // Secure omitted: app supports plain HTTP in dev
		Name:     AuthCookieName,
		Value:    tokens.AccessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  tokens.AccessExpiresAt,
		Secure:   h.config.IsProduction(),
	})
}
