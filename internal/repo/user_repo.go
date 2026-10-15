package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo is a Postgres-backed implementation of data.UserRepoer.
type UserRepo struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

// Ensure UserRepo implements data.UserRepoer.
var _ data.UserRepoer = (*UserRepo)(nil)

// NewUserRepo creates a new Postgres-backed user repository.
func NewUserRepo(q *db.Queries, pool *pgxpool.Pool) data.UserRepoer {
	return &UserRepo{q: q, pool: pool}
}

// Create inserts a new user, stamping ID/CreatedAt/UpdatedAt if unset.
func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	if user.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			id = uuid.New()
		}
		user.ID = id
	}
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now
	if user.PasswordChangedAt.IsZero() {
		user.PasswordChangedAt = now
	}

	role := int16(user.Role) //nolint:gosec // UserRole has 3 values, always fits int16
	created, err := r.q.CreateUser(ctx, db.CreateUserParams{
		ID:                user.ID,
		Email:             user.Email,
		Name:              user.Name,
		PasswordHash:      user.PasswordHash,
		Role:              role,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
	})
	if err != nil {
		return mapPostgresError(err)
	}
	*user = *toModelUser(&created)
	return nil
}

// GetByID retrieves a user by ID.
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	return toModelUser(&u), nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	return toModelUser(&u), nil
}

// UpdateRole changes a user's role.
func (r *UserRepo) UpdateRole(
	ctx context.Context,
	id uuid.UUID,
	role model.UserRole,
) (*model.User, error) {
	u, err := r.q.UpdateUserRole(ctx, db.UpdateUserRoleParams{
		ID:        id,
		Role:      int16(role), //nolint:gosec // UserRole has 3 values, always fits int16
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return nil, mapPostgresError(err)
	}
	return toModelUser(&u), nil
}

// UpdateProfile changes a user's name and email.
func (r *UserRepo) UpdateProfile(
	ctx context.Context,
	id uuid.UUID,
	name, email string,
) (*model.User, error) {
	u, err := r.q.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:        id,
		Name:      name,
		Email:     email,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return nil, mapPostgresError(err)
	}
	return toModelUser(&u), nil
}

// UpdatePassword stores a new password hash and stamps PasswordChangedAt.
func (r *UserRepo) UpdatePassword(
	ctx context.Context,
	id uuid.UUID,
	passwordHash string,
) (*model.User, error) {
	u, err := r.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:                id,
		PasswordHash:      passwordHash,
		PasswordChangedAt: time.Now(),
	})
	if err != nil {
		return nil, mapPostgresError(err)
	}
	return toModelUser(&u), nil
}

func toModelUser(u *db.User) *model.User {
	return &model.User{
		ID:                u.ID,
		Email:             u.Email,
		Name:              u.Name,
		PasswordHash:      u.PasswordHash,
		Role:              model.UserRole(u.Role),
		PasswordChangedAt: u.PasswordChangedAt,
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
	}
}
