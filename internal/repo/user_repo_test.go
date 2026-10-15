package repo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
)

func newTestUser(email string) *model.User {
	return &model.User{
		Email:        email,
		Name:         "Test User",
		PasswordHash: "not-a-real-hash",
		Role:         model.RoleUser,
	}
}

func TestUserRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("Create_StampsPasswordChangedAt", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			users := repo.NewUserRepo(q, pool)
			user := newTestUser(uuid.NewString() + "@example.com")
			if err := users.Create(ctx, user); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, err := users.GetByID(ctx, user.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.PasswordChangedAt.IsZero() || got.Email != user.Email {
				t.Errorf("unexpected user: %+v", got)
			}
		})
	})

	t.Run("UpdateProfile_ConflictsOnTakenEmail", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			users := repo.NewUserRepo(q, pool)
			a := newTestUser(uuid.NewString() + "@example.com")
			b := newTestUser(uuid.NewString() + "@example.com")
			for _, u := range []*model.User{a, b} {
				if err := users.Create(ctx, u); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
			updated, err := users.UpdateProfile(ctx, a.ID, "Renamed", a.Email)
			if err != nil || updated.Name != "Renamed" {
				t.Fatalf("expected rename to succeed, got %+v, %v", updated, err)
			}
			if _, err := users.UpdateProfile(ctx, a.ID, "Renamed", b.Email); !errors.Is(
				err, data.ErrConflict,
			) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})
	})

	t.Run("UpdateProfile_NotFound", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			users := repo.NewUserRepo(q, pool)
			_, err := users.UpdateProfile(ctx, uuid.New(), "Nobody", "nobody@example.com")
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	})

	t.Run("UpdatePassword_StampsPasswordChangedAt", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			users := repo.NewUserRepo(q, pool)
			user := newTestUser(uuid.NewString() + "@example.com")
			user.PasswordChangedAt = time.Now().Add(-24 * time.Hour)
			if err := users.Create(ctx, user); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			updated, err := users.UpdatePassword(ctx, user.ID, "new-hash")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if updated.PasswordHash != "new-hash" ||
				!updated.PasswordChangedAt.After(user.PasswordChangedAt) {
				t.Errorf("unexpected user: %+v", updated)
			}
		})
	})
}
