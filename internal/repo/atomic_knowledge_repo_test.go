package repo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
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
