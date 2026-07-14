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
	"github.com/pgvector/pgvector-go"
)

func newTestAtomicKnowledge(documentID, chunkID uuid.UUID) model.AtomicKnowledge {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return model.AtomicKnowledge{
		ID:              uuid.New(),
		CreatedAt:       now,
		UpdatedAt:       now,
		DocumentID:      documentID,
		DocumentChunkID: chunkID,
		TruthTier:       model.TruthTierAxiomatic,
		Topics:          []string{"topic-a"},
		Subject:         "Alice",
		Predicate:       "founded",
		Object:          "Acme Corp",
		Note:            "test fact",
	}
}

// TestAtomicKnowledgeRepo_WithTx exercises real transactional atomicity, so
// it cannot use testhelpers.WithTxRollback — AtomicKnowledgeRepo.WithTx opens
// its own transaction via pool.Begin, independent of any outer wrapper. Same
// reasoning as TestDocumentChunkRepo_WithTx_Enqueue.
func TestAtomicKnowledgeRepo_WithTx(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, q := testhelpers.SetupTestDB(t)
	t.Cleanup(func() { pool.Close() })

	ctx := t.Context()

	documentRepo := repo.NewDocumentRepo(q, pool, nil)
	doc := newTestDocument(uuid.NewString())
	if err := documentRepo.Create(ctx, doc); err != nil {
		t.Fatalf("unexpected error creating parent document: %v", err)
	}
	t.Cleanup(func() {
		// ON DELETE CASCADE takes any committed chunks/facts with it too.
		_, _ = pool.Exec(context.Background(), "delete from documents where id = $1", doc.ID)
	})

	chunkRepo := repo.NewDocumentChunkRepo(q, pool, nil)
	chunk := newTestDocumentChunk(doc.ID, 0)
	if err := chunkRepo.CreateBatch(ctx, []model.DocumentChunk{chunk}); err != nil {
		t.Fatalf("unexpected error creating parent chunk: %v", err)
	}

	knowledgeRepo := repo.NewAtomicKnowledgeRepo(q, pool)

	t.Run("Success_CommitsFactsAndMarksChunk", func(t *testing.T) {
		fact := newTestAtomicKnowledge(doc.ID, chunk.ID)

		err := knowledgeRepo.WithTx(ctx, func(tx data.AtomicKnowledgeRepoer) error {
			if err := tx.CreateBatch(ctx, []model.AtomicKnowledge{fact}); err != nil {
				return err
			}
			return tx.MarkChunkKnowledgeExtracted(ctx, chunk.ID)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var count int
		row := pool.QueryRow(
			ctx,
			"select count(*) from atomic_knowledge where document_chunk_id = $1",
			chunk.ID,
		)
		if err := row.Scan(&count); err != nil {
			t.Fatalf("unexpected error counting facts: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 fact persisted, got %d", count)
		}

		got, err := chunkRepo.GetByID(ctx, chunk.ID)
		if err != nil {
			t.Fatalf("unexpected error fetching chunk: %v", err)
		}
		if got.KnowledgeExtractedAt == nil {
			t.Error("expected KnowledgeExtractedAt to be set after extraction")
		}
	})

	t.Run("Failure_RollsBackBothFactsAndMark", func(t *testing.T) {
		chunk2 := newTestDocumentChunk(doc.ID, 1)
		if err := chunkRepo.CreateBatch(ctx, []model.DocumentChunk{chunk2}); err != nil {
			t.Fatalf("unexpected error creating second chunk: %v", err)
		}
		fact := newTestAtomicKnowledge(doc.ID, chunk2.ID)

		err := knowledgeRepo.WithTx(ctx, func(tx data.AtomicKnowledgeRepoer) error {
			if err := tx.CreateBatch(ctx, []model.AtomicKnowledge{fact}); err != nil {
				return err
			}
			return errors.New("boom")
		})
		if err == nil {
			t.Fatal("expected an error from WithTx")
		}

		var count int
		row := pool.QueryRow(
			ctx,
			"select count(*) from atomic_knowledge where document_chunk_id = $1",
			chunk2.ID,
		)
		if err := row.Scan(&count); err != nil {
			t.Fatalf("unexpected error counting facts: %v", err)
		}
		if count != 0 {
			t.Errorf("expected the fact insert to be rolled back, got %d rows", count)
		}

		got, err := chunkRepo.GetByID(ctx, chunk2.ID)
		if err != nil {
			t.Fatalf("unexpected error fetching chunk2: %v", err)
		}
		if got.KnowledgeExtractedAt != nil {
			t.Error("expected KnowledgeExtractedAt to remain nil after rollback")
		}
	})
}

func TestAtomicKnowledgeRepo_CreateBatch_Empty_NoOp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, q := testhelpers.SetupTestDB(t)
	defer pool.Close()

	knowledgeRepo := repo.NewAtomicKnowledgeRepo(q, pool)
	if err := knowledgeRepo.CreateBatch(t.Context(), nil); err != nil {
		t.Errorf("expected no error for an empty batch, got %v", err)
	}
}

func TestAtomicKnowledgeRepo_ListNeedingEmbedding_FiltersToUnembedded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	testhelpers.WithTxRollback(t, pool, func(ctx context.Context, q *db.Queries) {
		documentRepo := repo.NewDocumentRepo(q, pool, nil)
		doc := newTestDocument(uuid.NewString())
		if err := documentRepo.Create(ctx, doc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		chunkRepo := repo.NewDocumentChunkRepo(q, pool, nil)
		chunk := newTestDocumentChunk(doc.ID, 0)
		if err := chunkRepo.CreateBatch(ctx, []model.DocumentChunk{chunk}); err != nil {
			t.Fatalf("unexpected error creating chunk: %v", err)
		}

		knowledgeRepo := repo.NewAtomicKnowledgeRepo(q, pool)
		needsEmbedding := newTestAtomicKnowledge(doc.ID, chunk.ID)
		alreadyEmbedded := newTestAtomicKnowledge(doc.ID, chunk.ID)
		if err := knowledgeRepo.CreateBatch(
			ctx,
			[]model.AtomicKnowledge{needsEmbedding, alreadyEmbedded},
		); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := knowledgeRepo.UpdateEmbedding(
			ctx,
			alreadyEmbedded.ID,
			testEmbedding(0.2),
		); err != nil {
			t.Fatalf("unexpected error embedding fact: %v", err)
		}

		got, err := knowledgeRepo.ListNeedingEmbedding(ctx, doc.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected exactly 1 fact needing embedding, got %d: %+v", len(got), got)
		}
		if got[0].ID != needsEmbedding.ID {
			t.Errorf(
				"expected the unembedded fact %s, got %s",
				needsEmbedding.ID, got[0].ID,
			)
		}
	})
}

// TestAtomicKnowledgeRepo_UpdateEmbedding_PersistsAndRoundTrips verifies the
// stored vector via a raw query against pool, so it cannot use
// testhelpers.WithTxRollback — that leaves writes uncommitted, invisible to
// a query issued over a separate pool connection. Same reasoning as
// TestAtomicKnowledgeRepo_WithTx/TestDocumentChunkRepo_WithTx_Enqueue.
func TestAtomicKnowledgeRepo_UpdateEmbedding_PersistsAndRoundTrips(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, q := testhelpers.SetupTestDB(t)
	t.Cleanup(func() { pool.Close() })

	ctx := t.Context()

	documentRepo := repo.NewDocumentRepo(q, pool, nil)
	doc := newTestDocument(uuid.NewString())
	if err := documentRepo.Create(ctx, doc); err != nil {
		t.Fatalf("unexpected error creating parent document: %v", err)
	}
	t.Cleanup(func() {
		// ON DELETE CASCADE takes the chunk and fact with it too.
		_, _ = pool.Exec(context.Background(), "delete from documents where id = $1", doc.ID)
	})

	chunkRepo := repo.NewDocumentChunkRepo(q, pool, nil)
	chunk := newTestDocumentChunk(doc.ID, 0)
	if err := chunkRepo.CreateBatch(ctx, []model.DocumentChunk{chunk}); err != nil {
		t.Fatalf("unexpected error creating chunk: %v", err)
	}

	knowledgeRepo := repo.NewAtomicKnowledgeRepo(q, pool)
	fact := newTestAtomicKnowledge(doc.ID, chunk.ID)
	if err := knowledgeRepo.CreateBatch(ctx, []model.AtomicKnowledge{fact}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := testEmbedding(0.3)
	if err := knowledgeRepo.UpdateEmbedding(ctx, fact.ID, want); err != nil {
		t.Fatalf("unexpected error updating embedding: %v", err)
	}

	stillNeeding, err := knowledgeRepo.ListNeedingEmbedding(ctx, doc.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stillNeeding) != 0 {
		t.Fatalf(
			"expected the embedded fact to no longer need embedding, got %d: %+v",
			len(stillNeeding), stillNeeding,
		)
	}

	var roundTripped pgvector.Vector
	row := pool.QueryRow(ctx, "select embedding from atomic_knowledge where id = $1", fact.ID)
	if err := row.Scan(&roundTripped); err != nil {
		t.Fatalf("unexpected error reading back embedding: %v", err)
	}
	got := roundTripped.Slice()
	if len(got) != len(want) {
		t.Fatalf("expected embedding of length %d, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("expected embedding %v, got %v", want, got)
			break
		}
	}
}
