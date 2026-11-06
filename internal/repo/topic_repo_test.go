package repo_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// withRollbackTx runs fn in a transaction that is always rolled back, so the
// tests leave no rows behind.
func withRollbackTx(
	t *testing.T,
	pool *pgxpool.Pool,
	fn func(ctx context.Context, tx pgx.Tx, q *db.Queries),
) {
	t.Helper()

	ctx := t.Context()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin test tx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fn(ctx, tx, db.New(tx))
}

func insertTopic(
	ctx context.Context,
	t *testing.T,
	tx pgx.Tx,
	name string,
	active bool,
) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := tx.QueryRow(ctx,
		`insert into topics (name, is_active) values ($1, $2) returning id`,
		name, active,
	).Scan(&id)
	if err != nil {
		t.Fatalf("failed to insert topic %q: %v", name, err)
	}
	return id
}

// deactivateAllTopics makes the rows of the migrations inactive inside the
// test transaction, so a test controls which topics are active.
func deactivateAllTopics(ctx context.Context, t *testing.T, tx pgx.Tx) {
	t.Helper()

	if _, err := tx.Exec(ctx, `update topics set is_active = false`); err != nil {
		t.Fatalf("failed to deactivate topics: %v", err)
	}
}

func topicNames(items []model.Topic) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func TestTopicRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	newRepo := func(q *db.Queries) data.TopicRepoer {
		return repo.NewTopicRepo(q, pool)
	}

	t.Run("Create_FillsStoredFields", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			topics := newRepo(q)
			name := "zz-" + uuid.NewString()
			topic := &model.Topic{Name: name, Description: "Desc", IsActive: true}

			if err := topics.Create(ctx, topic); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if topic.ID == uuid.Nil || topic.CreatedAt.IsZero() || topic.UpdatedAt.IsZero() {
				t.Errorf("unexpected stored fields: %+v", topic)
			}

			got, err := topics.GetByID(ctx, topic.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != name || got.Description != "Desc" || !got.IsActive {
				t.Errorf("unexpected topic: %+v", got)
			}
		})
	})

	t.Run("Create_Inactive_IsStoredInactive", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			topics := newRepo(q)
			topic := &model.Topic{Name: "zz-" + uuid.NewString(), IsActive: false}

			if err := topics.Create(ctx, topic); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if topic.IsActive {
				t.Errorf("expected an inactive topic, got %+v", topic)
			}
		})
	})
	t.Run("Create_DuplicateName_Conflicts", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			name := "zz-" + uuid.NewString()
			insertTopic(ctx, t, tx, name, true)

			// Keep this last: a unique violation aborts the transaction.
			err := newRepo(q).Create(ctx, &model.Topic{Name: name, IsActive: true})
			if !errors.Is(err, data.ErrConflict) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			_, err := newRepo(q).GetByID(ctx, uuid.New())
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	})

	t.Run("Update_ReplacesEditableFields", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			id := insertTopic(ctx, t, tx, "zz-"+uuid.NewString(), true)
			insertTopic(ctx, t, tx, "zz-"+uuid.NewString(), true) // another active one
			topics := newRepo(q)

			newName := "zz-" + uuid.NewString()
			updated := &model.Topic{ID: id, Name: newName, Description: "New", IsActive: false}
			if err := topics.Update(ctx, updated); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if updated.Name != newName || updated.IsActive || updated.UpdatedAt.IsZero() {
				t.Errorf("unexpected stored fields: %+v", updated)
			}

			got, err := topics.GetByID(ctx, id)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != newName || got.Description != "New" || got.IsActive {
				t.Errorf("unexpected topic: %+v", got)
			}
		})
	})

	t.Run("Update_UnknownID_NotFound", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			err := newRepo(q).Update(ctx, &model.Topic{
				ID: uuid.New(), Name: "zz-" + uuid.NewString(), IsActive: true,
			})
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	})

	t.Run("Update_DeactivatingLastActiveTopic_NotFound", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			deactivateAllTopics(ctx, t, tx)
			name := "zz-" + uuid.NewString()
			id := insertTopic(ctx, t, tx, name, true)
			topics := newRepo(q)

			err := topics.Update(ctx, &model.Topic{ID: id, Name: name, IsActive: false})
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}

			got, err := topics.GetByID(ctx, id)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.IsActive {
				t.Errorf("expected the last active topic to stay active, got %+v", got)
			}
		})
	})

	t.Run("Update_LastActiveTopic_CanStayActiveAndBeRenamed", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			deactivateAllTopics(ctx, t, tx)
			id := insertTopic(ctx, t, tx, "zz-"+uuid.NewString(), true)

			newName := "zz-" + uuid.NewString()
			err := newRepo(q).Update(ctx, &model.Topic{ID: id, Name: newName, IsActive: true})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	t.Run("Update_InactiveTopic_CanStayInactive", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			deactivateAllTopics(ctx, t, tx)
			insertTopic(ctx, t, tx, "zz-"+uuid.NewString(), true)
			id := insertTopic(ctx, t, tx, "zz-"+uuid.NewString(), false)

			err := newRepo(q).Update(ctx, &model.Topic{
				ID: id, Name: "zz-" + uuid.NewString(), IsActive: false,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	t.Run("Update_DuplicateName_Conflicts", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			taken := "zz-" + uuid.NewString()
			insertTopic(ctx, t, tx, taken, true)
			id := insertTopic(ctx, t, tx, "zz-"+uuid.NewString(), true)

			// Keep this last: a unique violation aborts the transaction.
			err := newRepo(q).Update(ctx, &model.Topic{ID: id, Name: taken, IsActive: true})
			if !errors.Is(err, data.ErrConflict) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})
	})

	t.Run("Paginate_FiltersSortsAndPages", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			token := "zz" + uuid.NewString()[:8]
			insertTopic(ctx, t, tx, token+"-b", true)
			insertTopic(ctx, t, tx, token+"-a", true)
			insertTopic(ctx, t, tx, token+"-c", false)
			topics := newRepo(q)

			all, err := topics.Paginate(ctx, &dto.PaginationDTO{Page: 1, PerPage: 10},
				data.AdminListFilter{Search: token})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := []string{token + "-a", token + "-b", token + "-c"}
			if fmt.Sprint(topicNames(all.Items)) != fmt.Sprint(want) || all.TotalItems != 3 {
				t.Errorf("expected %v ordered by name, got %v (total %d)",
					want, topicNames(all.Items), all.TotalItems)
			}

			// Search ignores case.
			upper, err := topics.Paginate(ctx, &dto.PaginationDTO{Page: 1, PerPage: 10},
				data.AdminListFilter{Search: "ZZ" + token[2:]})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if upper.TotalItems != 3 {
				t.Errorf("expected a case-insensitive search to find 3, got %d", upper.TotalItems)
			}

			active, inactive := true, false
			onlyActive, err := topics.Paginate(ctx, &dto.PaginationDTO{Page: 1, PerPage: 10},
				data.AdminListFilter{Search: token, IsActive: &active})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if onlyActive.TotalItems != 2 || len(onlyActive.Items) != 2 {
				t.Errorf("expected 2 active topics, got %v", topicNames(onlyActive.Items))
			}

			onlyInactive, err := topics.Paginate(ctx, &dto.PaginationDTO{Page: 1, PerPage: 10},
				data.AdminListFilter{Search: token, IsActive: &inactive})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if onlyInactive.TotalItems != 1 || onlyInactive.Items[0].Name != token+"-c" {
				t.Errorf("expected only %s-c, got %v", token, topicNames(onlyInactive.Items))
			}

			page2, err := topics.Paginate(ctx, &dto.PaginationDTO{Page: 2, PerPage: 2},
				data.AdminListFilter{Search: token})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if page2.Page != 2 || page2.PerPage != 2 || page2.TotalItems != 3 ||
				len(page2.Items) != 1 || page2.Items[0].Name != token+"-c" {
				t.Errorf("unexpected second page: %+v", page2)
			}
		})
	})

	t.Run("Paginate_NoMatch_ReturnsEmptyItems", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			got, err := newRepo(q).Paginate(ctx,
				&dto.PaginationDTO{Page: 1, PerPage: 10},
				data.AdminListFilter{Search: "zz-" + uuid.NewString()})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Items == nil || len(got.Items) != 0 || got.TotalItems != 0 {
				t.Errorf("expected an empty, non-nil page, got %+v", got)
			}
		})
	})

	t.Run("Create_DuplicateNameIgnoringCase_Conflicts", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			name := "zz-Case-" + uuid.NewString()
			insertTopic(ctx, t, tx, name, true)

			err := newRepo(q).Create(ctx,
				&model.Topic{Name: strings.ToUpper(name), IsActive: true})
			if !errors.Is(err, data.ErrConflict) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})
	})

	t.Run("Update_RenameToNameIgnoringCase_Conflicts", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			taken := "zz-Case-" + uuid.NewString()
			insertTopic(ctx, t, tx, taken, true)
			id := insertTopic(ctx, t, tx, "zz-"+uuid.NewString(), true)

			err := newRepo(q).Update(ctx,
				&model.Topic{ID: id, Name: strings.ToUpper(taken), IsActive: true})
			if !errors.Is(err, data.ErrConflict) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})
	})
}
