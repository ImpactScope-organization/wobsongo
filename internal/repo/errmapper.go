package repo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MapPostgresError translates a low-level pgx/PostgreSQL error into a high-level domain error.
// The returned domain error is safe to return directly to the Service layer.
func MapPostgresError(err error) error {
	if err == nil {
		return nil
	}

	// 1. Check for standard pgx errors
	if errors.Is(err, pgx.ErrNoRows) {
		// Example: SELECT failed to return a row
		return ErrNotFound
	}

	// 2. Check for specific PostgreSQL constraint errors
	// pgx wraps the underlying database error in a *pgconn.PgError
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// See: https://www.postgresql.org/docs/current/errcodes-appendix.html
		switch pgErr.Code {
		case "23505": // unique_violation (e.g., trying to insert a user with an existing unique email/username)
			// Check which constraint was violated to return more specific errors
			if strings.Contains(pgErr.ConstraintName, "username") {
				return ErrUsernameTaken
			}
			// For other unique violations (email, etc.), return generic conflict error
			return fmt.Errorf("%w: %s", ErrConflict, pgErr.Detail)

		case "23503": // foreign_key_violation (e.g., inserting a post with a non-existent user_id)
			return fmt.Errorf("%w: %s", ErrForeignKey, pgErr.Detail)

		case "23502": // not_null_violation (e.g., trying to insert NULL into a NOT NULL column)
			return fmt.Errorf("%w: not null field missing: %s", ErrInvalidInput, pgErr.Detail)

		case "22001": // string_data_length_mismatch (e.g., input too long for a VARCHAR column)
			return fmt.Errorf("%w: data length mismatch: %s", ErrInvalidInput, pgErr.Detail)

		// 42P01: undefined_table - usually a code bug, but worth checking.
		// 42501: insufficient_privilege - usually a configuration error.

		default:
			// For all other specific codes, treat them as an internal failure
			// Wrap the original error to preserve the stack trace and context
			return fmt.Errorf("%w: code %s, msg: %s", ErrInternalDBFail, pgErr.Code, pgErr.Message)
		}
	}

	// 3. Handle general connection or unexpected pgx/DB errors
	// This covers network timeouts, connection refused, or errors not wrapped by *pgconn.PgError
	return fmt.Errorf("%w: unexpected error: %s", ErrInternalDBFail, err.Error())
}
