package repo

import "errors"

var (
	ErrNotFound                  = errors.New("core: resource not found")
	ErrConflict                  = errors.New("core: resource conflict (already exists)")
	ErrForeignKey                = errors.New("core: foreign key constraint violation")
	ErrInvalidInput              = errors.New("core: input data validation failed")
	ErrInternalDBFail            = errors.New("core: internal database failure")
	ErrInvalidMimeType           = errors.New("invalid mime type")
	ErrMimeTypeMismatch          = errors.New("mime type mismatch")
	ErrInvalidToken              = errors.New("core: invalid token")
	ErrExpiredToken              = errors.New("core: expired token")
	ErrUnauthorized              = errors.New("core: unauthorized - incorrect credentials")
	ErrUsernameGeneration        = errors.New("failed to generate a unique username")
	ErrUsernameTaken             = errors.New("username is already taken")
	ErrTeamMaxMemberCountReached = errors.New("team has reached maximum member count")
	ErrActivityTypeMismatch      = errors.New(
		"activity type mismatch: operation not allowed for this activity type",
	)
	ErrEmptyMediaKey            = errors.New("media key cannot be empty")
	ErrInvalidMalformedMediaKey = errors.New("invalid or malformed media key")
	ErrTournamentStartAfterEnd  = errors.New(
		"tournament start date must be before end date",
	)
	ErrTournamentDatesOutsideSeason = errors.New(
		"tournament dates must fall within the season date range",
	)
	ErrActivityNotReusable = errors.New(
		"activity is not marked as reusable in non-always-on tournaments",
	)
	ErrForbidden          = errors.New("core: forbidden - insufficient permissions")
	ErrApprovedSubmission = errors.New(
		"core: submission is already approved and cannot be deleted",
	)
)
