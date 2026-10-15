package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

var msgPasswordPolicy = fmt.Sprintf(
	"password must be at least %d characters and at most %d bytes",
	auth.PasswordMinLength, auth.PasswordMaxBytes,
)

// AuthHandler serves sign-in and the signed-in user's own account as JSON.
type AuthHandler struct {
	service *service.AuthService
}

// NewAuthHandler creates a new handler.
func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// @Summary		Log in
// @Description	Exchanges an email and password for a token pair. Emails are case-insensitive. A wrong password and an unknown email give the same 401.
// @Tags		auth
// @Accept		json
// @Produce		json
// @Param		form	body		dto.LoginRequest	true	"Credentials"
// @Success		200		{object}	model.APIResponse{data=dto.TokenPairResponse}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		429		{object}	model.APIResponse{error=string}
// @Router		/api/v1/auth/login [post]
func (h *AuthHandler) loginHandler(c echo.Context) error {
	req := new(dto.LoginRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	tokens, err := h.service.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return authError(err)
	}
	return writeJSON(c, http.StatusOK, toTokenPairResponse(tokens))
}

// @Summary		Sign up
// @Description	Creates an account with the "user" role and signs it in.
// @Tags		auth
// @Accept		json
// @Produce		json
// @Param		form	body		dto.SignupRequest	true	"New account"
// @Success		201		{object}	model.APIResponse{data=dto.TokenPairResponse}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		409		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		429		{object}	model.APIResponse{error=string}
// @Router		/api/v1/auth/signup [post]
func (h *AuthHandler) signupHandler(c echo.Context) error {
	req := new(dto.SignupRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	if !auth.PasswordValid(req.Password) {
		return &model.APIError{
			Code:     http.StatusUnprocessableEntity,
			Internal: errors.New("password policy"),
			Public:   msgPasswordPolicy,
		}
	}
	_, tokens, err := h.service.Register(
		c.Request().Context(), req.Name, req.Email, req.Password,
	)
	if errors.Is(err, data.ErrConflict) {
		return &model.APIError{
			Code:     http.StatusConflict,
			Internal: err,
			Public:   "an account with this email already exists",
		}
	}
	if err != nil {
		return authError(err)
	}
	return writeJSON(c, http.StatusCreated, toTokenPairResponse(tokens))
}

// @Summary		Refresh tokens
// @Description	Exchanges a refresh token for a new token pair with the user's current name and role. Refresh tokens are stateless: they stay valid until they expire, unless the password changes.
// @Tags		auth
// @Accept		json
// @Produce		json
// @Param		form	body		dto.RefreshRequest	true	"Refresh token"
// @Success		200		{object}	model.APIResponse{data=dto.TokenPairResponse}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		429		{object}	model.APIResponse{error=string}
// @Router		/api/v1/auth/refresh [post]
func (h *AuthHandler) refreshHandler(c echo.Context) error {
	req := new(dto.RefreshRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	tokens, err := h.service.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return authError(err)
	}
	return writeJSON(c, http.StatusOK, toTokenPairResponse(tokens))
}

// @Summary		Get my account
// @Description	The signed-in user's own account.
// @Tags		me
// @Produce		json
// @Security	BearerAuth
// @Success		200		{object}	model.APIResponse{data=dto.MeResponse}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Router		/api/v1/me [get]
func (h *AuthHandler) meHandler(c echo.Context) error {
	user, err := h.service.Me(c.Request().Context(), ClaimsFromContext(c).UserID)
	if err != nil {
		return authError(err)
	}
	return writeJSON(c, http.StatusOK, toMeResponse(user))
}

// @Summary		Update my profile
// @Description	Changes the signed-in user's name and email.
// @Tags		me
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		form	body		dto.UpdateProfileRequest	true	"Profile"
// @Success		200		{object}	model.APIResponse{data=dto.MeResponse}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		409		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Router		/api/v1/me [patch]
func (h *AuthHandler) updateProfileHandler(c echo.Context) error {
	req := new(dto.UpdateProfileRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	user, err := h.service.UpdateProfile(
		c.Request().Context(), ClaimsFromContext(c).UserID, req.Name, req.Email,
	)
	if errors.Is(err, data.ErrConflict) {
		return &model.APIError{
			Code:     http.StatusConflict,
			Internal: err,
			Public:   "another account uses this email",
		}
	}
	if err != nil {
		return authError(err)
	}
	return writeJSON(c, http.StatusOK, toMeResponse(user))
}

// @Summary		Change my password
// @Description	Sets a new password after checking the current one. Returns a new token pair; refresh tokens issued before the change stop working, which signs out other devices.
// @Tags		me
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		form	body		dto.ChangePasswordRequest	true	"Passwords"
// @Success		200		{object}	model.APIResponse{data=dto.TokenPairResponse}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		403		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Router		/api/v1/me/password [post]
func (h *AuthHandler) changePasswordHandler(c echo.Context) error {
	req := new(dto.ChangePasswordRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	if !auth.PasswordValid(req.NewPassword) {
		return &model.APIError{
			Code:     http.StatusUnprocessableEntity,
			Internal: errors.New("password policy"),
			Public:   msgPasswordPolicy,
		}
	}
	tokens, err := h.service.ChangePassword(
		c.Request().Context(), ClaimsFromContext(c).UserID, req.CurrentPassword, req.NewPassword,
	)
	if errors.Is(err, auth.ErrIncorrectCredentials) {
		// 403, not 401: the caller is signed in, the password they typed is wrong.
		return &model.APIError{
			Code:     http.StatusForbidden,
			Internal: err,
			Public:   "current password is incorrect",
		}
	}
	if err != nil {
		return authError(err)
	}
	return writeJSON(c, http.StatusOK, toTokenPairResponse(tokens))
}

func bindAndValidate(c echo.Context, req any) error {
	if err := c.Bind(req); err != nil {
		return &model.APIError{
			Code:     http.StatusBadRequest,
			Internal: err,
			Public:   msgInvalidRequestBody,
		}
	}
	if err := c.Validate(req); err != nil {
		return &model.APIError{
			Code:     http.StatusUnprocessableEntity,
			Internal: err,
			Public:   msgValidationFailed,
		}
	}
	return nil
}

// authError maps auth and data errors: bad credentials and any token problem
// are 401, a user deleted since the token was issued is 401 too.
func authError(err error) error {
	switch {
	case errors.Is(err, auth.ErrIncorrectCredentials):
		return &model.APIError{
			Code:     http.StatusUnauthorized,
			Internal: err,
			Public:   "incorrect email or password",
		}
	case errors.Is(err, auth.ErrTokenExpired),
		errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrWrongTokenType),
		errors.Is(err, data.ErrNotFound):
		return &model.APIError{
			Code:     http.StatusUnauthorized,
			Internal: err,
			Public:   "invalid or expired token",
		}
	}
	status, message := mapDataErrorToHTTP(err)
	return &model.APIError{Code: status, Internal: err, Public: message}
}

func toTokenPairResponse(t *auth.JWTTokens) dto.TokenPairResponse {
	return dto.TokenPairResponse{
		AccessToken:      t.AccessToken,
		AccessExpiresAt:  t.AccessExpiresAt,
		RefreshToken:     t.RefreshToken,
		RefreshExpiresAt: t.RefreshExpiresAt,
	}
}

func toMeResponse(u *model.User) dto.MeResponse {
	return dto.MeResponse{
		ID:                u.ID,
		Name:              u.Name,
		Email:             u.Email,
		Role:              u.Role.String(),
		PasswordChangedAt: u.PasswordChangedAt,
	}
}
