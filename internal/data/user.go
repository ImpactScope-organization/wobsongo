package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// UserRepoer defines the interface for interacting with user data storage.
// Deliberately not CrudableWithTx — the web layer's auth flows only need
// create/lookup/profile/password/role updates, not pagination, generic Update, or Delete.
type UserRepoer interface {
	// Create inserts a new user. Returns ErrConflict if the email is
	// already registered.
	Create(ctx context.Context, user *model.User) error

	// GetByID retrieves a user by ID. Returns ErrNotFound if none exists.
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)

	// GetByEmail retrieves a user by email. Returns ErrNotFound if none exists.
	GetByEmail(ctx context.Context, email string) (*model.User, error)

	// UpdateRole changes a user's role. Returns ErrNotFound if no user has that ID.
	UpdateRole(ctx context.Context, id uuid.UUID, role model.UserRole) (*model.User, error)

	// UpdateProfile changes a user's name and email. Returns ErrNotFound if
	// no user has that ID, ErrConflict if another user has the email.
	UpdateProfile(ctx context.Context, id uuid.UUID, name, email string) (*model.User, error)

	// UpdatePassword stores a new password hash and stamps
	// PasswordChangedAt. Returns ErrNotFound if no user has that ID.
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) (*model.User, error)
}
