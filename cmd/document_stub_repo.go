package cmd

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// newStubDocumentRepo returns a temporary in-memory-backed data.DocumentRepoer,
// built on the moq-generated mock, standing in until a real Postgres-backed
// repo (internal/repo/document_repo.go) exists. Replace this wiring once that lands.
// Job enqueueing is genuinely real (backed by riverClient) even though document
// persistence itself stays in-memory.
func newStubDocumentRepo(riverClient *river.Client[pgx.Tx]) data.DocumentRepoer {
	var mu sync.Mutex
	store := make(map[uuid.UUID]*model.Document)

	mock := &mockrepo.DocumentRepoerMock{}

	mock.CreateFunc = func(_ context.Context, entity *model.Document) error {
		mu.Lock()
		defer mu.Unlock()
		store[entity.ID] = entity
		return nil
	}

	mock.GetByIDFunc = func(_ context.Context, id uuid.UUID) (*model.Document, error) {
		mu.Lock()
		defer mu.Unlock()
		doc, ok := store[id]
		if !ok {
			return nil, data.ErrNotFound
		}
		return doc, nil
	}

	mock.UpdateFunc = func(_ context.Context, entity *model.Document) error {
		mu.Lock()
		defer mu.Unlock()
		if _, ok := store[entity.ID]; !ok {
			return data.ErrNotFound
		}
		store[entity.ID] = entity
		return nil
	}

	mock.DeleteFunc = func(_ context.Context, id uuid.UUID) error {
		mu.Lock()
		defer mu.Unlock()
		if _, ok := store[id]; !ok {
			return data.ErrNotFound
		}
		delete(store, id)
		return nil
	}

	mock.PaginateFunc = func(
		_ context.Context,
		_ data.SupportsPagination,
	) (*dto.PaginationResults[model.Document], error) {
		mu.Lock()
		defer mu.Unlock()
		items := make([]model.Document, 0, len(store))
		for _, doc := range store {
			items = append(items, *doc)
		}
		return &dto.PaginationResults[model.Document]{
			Page:       1,
			PerPage:    len(items),
			TotalItems: len(items),
			Items:      items,
		}, nil
	}

	mock.WithTxFunc = func(_ context.Context, fn func(data.DocumentRepoer) error) error {
		return fn(mock)
	}

	mock.EnqueueFunc = func(ctx context.Context, payload queue.BackgroundJob) error {
		_, err := riverClient.Insert(ctx, payload, nil)
		return err
	}

	return mock
}
