package repo_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func newTestDocumentChunk(documentID uuid.UUID, seq int) model.DocumentChunk {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return model.DocumentChunk{
		ID:              uuid.New(),
		CreatedAt:       now,
		UpdatedAt:       now,
		DocumentID:      documentID,
		SequenceNumber:  seq,
		Topics:          []string{"topic-a"},
		FactualityScore: 0.5,
		ParsedChunk: model.ParsedChunk{
			Text:        "chunk text",
			Page:        1,
			LayoutType:  model.LayoutTypeParagraph,
			BoundingBox: model.BoundingBox{1, 2, 3, 4},
		},
	}
}

func TestDocumentChunkRepo_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("CreateBatch_And_ListByDocumentID", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			documentRepo := repo.NewDocumentRepo(q, pool, nil)
			doc := newTestDocument(uuid.NewString())
			if err := documentRepo.Create(ctx, doc); err != nil {
				t.Fatalf("unexpected error creating parent document: %v", err)
			}

			chunkRepo := repo.NewDocumentChunkRepo(q, pool, nil)
			chunks := []model.DocumentChunk{
				newTestDocumentChunk(doc.ID, 0),
				newTestDocumentChunk(doc.ID, 1),
			}
			if err := chunkRepo.CreateBatch(ctx, chunks); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := chunkRepo.ListByDocumentID(ctx, doc.ID)
			if err != nil {
				t.Fatalf("unexpected error listing chunks: %v", err)
			}
			if len(got) != 2 {
				t.Fatalf("expected 2 chunks, got %d", len(got))
			}
			if got[0].SequenceNumber != 0 || got[1].SequenceNumber != 1 {
				t.Errorf("expected chunks ordered by sequence number, got %+v", got)
			}
		})
	})

	t.Run("CreateBatch_Empty_NoOp", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			chunkRepo := repo.NewDocumentChunkRepo(q, pool, nil)
			if err := chunkRepo.CreateBatch(ctx, nil); err != nil {
				t.Errorf("expected no error for an empty batch, got %v", err)
			}
		})
	})

	t.Run("GetByID_Success", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			documentRepo := repo.NewDocumentRepo(q, pool, nil)
			doc := newTestDocument(uuid.NewString())
			if err := documentRepo.Create(ctx, doc); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			chunkRepo := repo.NewDocumentChunkRepo(q, pool, nil)
			chunk := newTestDocumentChunk(doc.ID, 0)
			if err := chunkRepo.CreateBatch(ctx, []model.DocumentChunk{chunk}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := chunkRepo.GetByID(ctx, chunk.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Text != chunk.Text || got.BoundingBox != chunk.BoundingBox {
				t.Errorf("fetched chunk does not match created one: %+v vs %+v", got, chunk)
			}
		})
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			chunkRepo := repo.NewDocumentChunkRepo(q, pool, nil)
			_, err := chunkRepo.GetByID(ctx, uuid.New())
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected data.ErrNotFound, got %v", err)
			}
		})
	})

	t.Run("Update_Success", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			documentRepo := repo.NewDocumentRepo(q, pool, nil)
			doc := newTestDocument(uuid.NewString())
			if err := documentRepo.Create(ctx, doc); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			chunkRepo := repo.NewDocumentChunkRepo(q, pool, nil)
			chunk := newTestDocumentChunk(doc.ID, 0)
			if err := chunkRepo.CreateBatch(ctx, []model.DocumentChunk{chunk}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			chunk.Text = "updated text"
			chunk.Topics = []string{"updated-topic"}
			chunk.FactualityScore = 0.9
			if err := chunkRepo.Update(ctx, &chunk); err != nil {
				t.Fatalf("unexpected error updating chunk: %v", err)
			}

			got, err := chunkRepo.GetByID(ctx, chunk.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Text != "updated text" || got.FactualityScore != 0.9 {
				t.Errorf("update did not persist: %+v", got)
			}
		})
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			chunkRepo := repo.NewDocumentChunkRepo(q, pool, nil)
			chunk := newTestDocumentChunk(uuid.New(), 0)
			err := chunkRepo.Update(ctx, &chunk)
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected data.ErrNotFound, got %v", err)
			}
		})
	})

	t.Run("ShouldBeStored_PassThrough", func(t *testing.T) {
		testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
			chunkRepo := repo.NewDocumentChunkRepo(q, pool, nil)
			ok, err := chunkRepo.ShouldBeStored(ctx, newTestDocumentChunk(uuid.New(), 0))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ok {
				t.Error("expected the pass-through ShouldBeStored to always return true")
			}
		})
	})
}

// TestDocumentChunkRepo_WithTx_Enqueue exercises real transactional
// atomicity, so it cannot use testhelpers.WithTxRollback — see the identical
// reasoning in TestDocumentRepo_WithTx_Enqueue.
func TestDocumentChunkRepo_WithTx_Enqueue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, q := testhelpers.SetupTestDB(t)
	// t.Cleanup (not defer): cleanups run in LIFO order after the test body
	// completes, so registering the pool close first and the row delete
	// second guarantees the delete runs while the pool is still open — a
	// plain `defer pool.Close()` here would fire before this delete cleanup
	// and silently no-op it (a real bug caught while writing this test).
	t.Cleanup(func() { pool.Close() })

	ctx := t.Context()

	documentRepo := repo.NewDocumentRepo(q, pool, nil)
	doc := newTestDocument(uuid.NewString())
	if err := documentRepo.Create(ctx, doc); err != nil {
		t.Fatalf("unexpected error creating parent document: %v", err)
	}
	t.Cleanup(func() {
		// ON DELETE CASCADE takes any committed chunks with it too.
		_, _ = pool.Exec(context.Background(), "delete from documents where id = $1", doc.ID)
	})

	workers := river.NewWorkers()
	river.AddWorker(workers, &noopParseDocumentWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("failed to construct river client: %v", err)
	}

	chunkRepo := repo.NewDocumentChunkRepo(
		q,
		pool,
		func() *river.Client[pgx.Tx] { return riverClient },
	)

	t.Run("Success_CommitsChunkAndEnqueuesJob", func(t *testing.T) {
		chunk := newTestDocumentChunk(doc.ID, 0)

		err := chunkRepo.WithTx(ctx, func(tx data.DocumentChunkRepoer) error {
			if err := tx.CreateBatch(ctx, []model.DocumentChunk{chunk}); err != nil {
				return err
			}
			return tx.Enqueue(ctx, queue.ParseDocumentDTO{
				DocumentID: doc.ID,
				FileKey:    string(doc.FileURL),
			})
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := chunkRepo.GetByID(ctx, chunk.ID); err != nil {
			t.Errorf("expected chunk to be committed, got error: %v", err)
		}

		row := pool.QueryRow(
			ctx,
			"select args from river_job where kind = $1 order by id desc limit 1",
			queue.JobTypeParseDocument,
		)
		var rawArgs []byte
		if err := row.Scan(&rawArgs); err != nil {
			t.Fatalf("expected an enqueued river_job row, got error: %v", err)
		}
		var args queue.ParseDocumentDTO
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			t.Fatalf("failed to unmarshal river_job args: %v", err)
		}
		if args.DocumentID != doc.ID {
			t.Errorf("expected enqueued job for document %s, got %s", doc.ID, args.DocumentID)
		}
	})

	t.Run("Failure_RollsBackChunkCreate", func(t *testing.T) {
		chunk := newTestDocumentChunk(doc.ID, 1)

		err := chunkRepo.WithTx(ctx, func(tx data.DocumentChunkRepoer) error {
			if err := tx.CreateBatch(ctx, []model.DocumentChunk{chunk}); err != nil {
				return err
			}
			return errors.New("boom")
		})
		if err == nil {
			t.Fatal("expected an error from WithTx")
		}

		if _, err := chunkRepo.GetByID(ctx, chunk.ID); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("expected the chunk create to be rolled back, got: %v", err)
		}
	})
}
