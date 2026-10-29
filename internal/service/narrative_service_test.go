package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

const testMatchThreshold = 0.8

// narrativeStubEmbedder is a hand-rolled data.Embedder that returns the given
// vectors in order, one per embedded text.
type narrativeStubEmbedder struct {
	vectors [][]float32
	err     error
}

func (s *narrativeStubEmbedder) Embed(_ context.Context, _ []string) ([][]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.vectors, nil
}

// stubStatementGenerator is a hand-rolled data.NarrativeStatementGenerator.
// calls records every claim it was asked to write a statement for.
type stubStatementGenerator struct {
	statement string
	err       error
	calls     []string
}

func (s *stubStatementGenerator) Generate(_ context.Context, claim string) (string, error) {
	s.calls = append(s.calls, claim)
	if s.err != nil {
		return "", s.err
	}
	return s.statement, nil
}

// newNarrativeRepoMock returns a mock whose WithTx runs fn against the mock
// itself, so calls made inside the transaction are recorded on it.
func newNarrativeRepoMock() *mockrepo.NarrativeRepoerMock {
	repo := &mockrepo.NarrativeRepoerMock{}
	repo.WithTxFunc = func(_ context.Context, fn func(data.NarrativeRepoer) error) error {
		return fn(repo)
	}
	return repo
}

// clusterFixture wires a mock repo for the happy path of ClusterClaim. Tests
// override single funcs on repo to simulate a failure.
type clusterFixture struct {
	repo      *mockrepo.NarrativeRepoerMock
	generator *stubStatementGenerator
	claim     *model.Claim
}

func newClusterFixture(existing []model.Narrative) *clusterFixture {
	claim := &model.Claim{
		ID:       uuid.New(),
		TopicID:  uuid.New(),
		Text:     "vitamin C cures colds",
		Severity: model.SeveritySerious,
		Verdict:  model.VerdictContradicted,
	}

	repo := newNarrativeRepoMock()
	repo.GetClaimFunc = func(context.Context, uuid.UUID) (*model.Claim, error) {
		return claim, nil
	}
	repo.LockTopicFunc = func(context.Context, uuid.UUID) error { return nil }
	repo.ListNarrativesByTopicFunc = func(context.Context, uuid.UUID) ([]model.Narrative, error) {
		return existing, nil
	}
	repo.CreateNarrativeFunc = func(_ context.Context, n *model.Narrative) (*model.Narrative, error) {
		created := *n
		created.ID = uuid.New()
		return &created, nil
	}
	repo.LinkClaimFunc = func(context.Context, uuid.UUID, uuid.UUID) error { return nil }
	repo.GetNewestClaimFunc = func(context.Context, uuid.UUID) (*model.Claim, error) {
		return claim, nil
	}
	repo.UpdateNarrativeEstimateFunc = func(
		context.Context, uuid.UUID, *int, model.NarrativeStatus,
	) error {
		return nil
	}

	return &clusterFixture{
		repo:      repo,
		generator: &stubStatementGenerator{statement: "Vitamins cure illnesses"},
		claim:     claim,
	}
}

func TestNarrativeService_ClusterClaim(t *testing.T) {
	t.Run("StartsNewNarrativeWhenTopicHasNone", func(t *testing.T) {
		f := newClusterFixture(nil)
		s := NewNarrativeService(f.repo, &narrativeStubEmbedder{}, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		created := f.repo.CreateNarrativeCalls()
		if len(created) != 1 {
			t.Fatalf("expected 1 narrative created, got %d", len(created))
		}
		if created[0].Narrative.TopicID != f.claim.TopicID {
			t.Errorf("expected topic %s, got %s", f.claim.TopicID, created[0].Narrative.TopicID)
		}
		if created[0].Narrative.Statement != "Vitamins cure illnesses" {
			t.Errorf("expected generated statement, got %q", created[0].Narrative.Statement)
		}
		if len(f.generator.calls) != 1 || f.generator.calls[0] != f.claim.Text {
			t.Errorf("expected the generator to get the claim text, got %v", f.generator.calls)
		}

		linked := f.repo.LinkClaimCalls()
		if len(linked) != 1 || linked[0].ClaimID != f.claim.ID {
			t.Fatalf("expected the claim to be linked once, got %+v", linked)
		}

		updates := f.repo.UpdateNarrativeEstimateCalls()
		if len(updates) != 1 {
			t.Fatalf("expected 1 estimate update, got %d", len(updates))
		}
		if updates[0].NarrativeID != linked[0].NarrativeID {
			t.Errorf("expected the estimate to update the linked narrative")
		}
		if updates[0].Harm == nil || *updates[0].Harm != 3 {
			t.Errorf("expected harm 3 for a serious claim, got %v", updates[0].Harm)
		}
		if updates[0].Status != model.NarrativeStatusAutoChecked {
			t.Errorf(
				"expected status %q, got %q",
				model.NarrativeStatusAutoChecked,
				updates[0].Status,
			)
		}
	})

	t.Run("JoinsMostSimilarNarrative", func(t *testing.T) {
		far := model.Narrative{ID: uuid.New(), Statement: "unrelated narrative"}
		close := model.Narrative{ID: uuid.New(), Statement: "similar narrative"}
		f := newClusterFixture([]model.Narrative{far, close})
		// Vectors are claim, far, close: the claim points the same way as close.
		embedder := &narrativeStubEmbedder{vectors: [][]float32{{1, 0}, {0, 1}, {1, 0}}}
		s := NewNarrativeService(f.repo, embedder, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n := len(f.repo.CreateNarrativeCalls()); n != 0 {
			t.Errorf("expected no new narrative, got %d", n)
		}
		if len(f.generator.calls) != 0 {
			t.Errorf("expected no statement to be generated, got %v", f.generator.calls)
		}
		linked := f.repo.LinkClaimCalls()
		if len(linked) != 1 || linked[0].NarrativeID != close.ID {
			t.Errorf("expected the claim to join narrative %s, got %+v", close.ID, linked)
		}
	})

	t.Run("StartsNewNarrativeBelowThreshold", func(t *testing.T) {
		other := model.Narrative{ID: uuid.New(), Statement: "unrelated narrative"}
		f := newClusterFixture([]model.Narrative{other})
		embedder := &narrativeStubEmbedder{vectors: [][]float32{{1, 0}, {0, 1}}}
		s := NewNarrativeService(f.repo, embedder, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n := len(f.repo.CreateNarrativeCalls()); n != 1 {
			t.Errorf("expected a new narrative below the threshold, got %d", n)
		}
		linked := f.repo.LinkClaimCalls()
		if len(linked) != 1 || linked[0].NarrativeID == other.ID {
			t.Errorf("expected the claim to join the new narrative, got %+v", linked)
		}
	})

	t.Run("GetClaimError", func(t *testing.T) {
		f := newClusterFixture(nil)
		f.repo.GetClaimFunc = func(context.Context, uuid.UUID) (*model.Claim, error) {
			return nil, data.ErrNotFound
		}
		s := NewNarrativeService(f.repo, &narrativeStubEmbedder{}, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("expected %v, got %v", data.ErrNotFound, err)
		}
		if n := len(f.repo.WithTxCalls()); n != 0 {
			t.Errorf("expected no transaction without a claim, got %d", n)
		}
	})

	t.Run("LockTopicError", func(t *testing.T) {
		f := newClusterFixture(nil)
		f.repo.LockTopicFunc = func(context.Context, uuid.UUID) error { return data.ErrInternal }
		s := NewNarrativeService(f.repo, &narrativeStubEmbedder{}, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}
		if n := len(f.repo.ListNarrativesByTopicCalls()); n != 0 {
			t.Errorf("expected narratives not to be listed after a failed lock, got %d", n)
		}
	})

	t.Run("ListNarrativesError", func(t *testing.T) {
		f := newClusterFixture(nil)
		f.repo.ListNarrativesByTopicFunc = func(context.Context, uuid.UUID) ([]model.Narrative, error) {
			return nil, data.ErrInternal
		}
		s := NewNarrativeService(f.repo, &narrativeStubEmbedder{}, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}
	})

	t.Run("EmbedderError", func(t *testing.T) {
		wantErr := errors.New("embedder down")
		other := model.Narrative{ID: uuid.New(), Statement: "other"}
		f := newClusterFixture([]model.Narrative{other})
		s := NewNarrativeService(
			f.repo,
			&narrativeStubEmbedder{err: wantErr},
			f.generator,
			testMatchThreshold,
		)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if n := len(f.repo.LinkClaimCalls()); n != 0 {
			t.Errorf("expected the claim not to be linked, got %d link(s)", n)
		}
	})

	t.Run("EmbedderReturnsWrongVectorCount", func(t *testing.T) {
		other := model.Narrative{ID: uuid.New(), Statement: "other"}
		f := newClusterFixture([]model.Narrative{other})
		// Two texts are embedded (claim + 1 narrative) but one vector comes back.
		embedder := &narrativeStubEmbedder{vectors: [][]float32{{1, 0}}}
		s := NewNarrativeService(f.repo, embedder, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if got := fmt.Sprint(
			err,
		); !strings.Contains(
			got,
			"embedder returned 1 vectors for 2 texts",
		) {
			t.Fatalf("unexpected error: %v", err)
		}
		if n := len(f.repo.LinkClaimCalls()); n != 0 {
			t.Errorf("expected the claim not to be linked, got %d link(s)", n)
		}
	})

	t.Run("GeneratorError", func(t *testing.T) {
		wantErr := errors.New("llm down")
		f := newClusterFixture(nil)
		f.generator.err = wantErr
		s := NewNarrativeService(f.repo, &narrativeStubEmbedder{}, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if n := len(f.repo.CreateNarrativeCalls()); n != 0 {
			t.Errorf("expected no narrative without a statement, got %d", n)
		}
	})

	t.Run("CreateNarrativeError", func(t *testing.T) {
		f := newClusterFixture(nil)
		f.repo.CreateNarrativeFunc = func(context.Context, *model.Narrative) (*model.Narrative, error) {
			return nil, data.ErrInternal
		}
		s := NewNarrativeService(f.repo, &narrativeStubEmbedder{}, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}
		if n := len(f.repo.LinkClaimCalls()); n != 0 {
			t.Errorf("expected the claim not to be linked, got %d link(s)", n)
		}
	})

	t.Run("LinkClaimError", func(t *testing.T) {
		f := newClusterFixture(nil)
		f.repo.LinkClaimFunc = func(context.Context, uuid.UUID, uuid.UUID) error {
			return data.ErrInternal
		}
		s := NewNarrativeService(f.repo, &narrativeStubEmbedder{}, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}
		if n := len(f.repo.UpdateNarrativeEstimateCalls()); n != 0 {
			t.Errorf("expected no estimate update after a failed link, got %d", n)
		}
	})

	t.Run("GetNewestClaimError", func(t *testing.T) {
		f := newClusterFixture(nil)
		f.repo.GetNewestClaimFunc = func(context.Context, uuid.UUID) (*model.Claim, error) {
			return nil, data.ErrNotFound
		}
		s := NewNarrativeService(f.repo, &narrativeStubEmbedder{}, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("expected %v, got %v", data.ErrNotFound, err)
		}
		if n := len(f.repo.UpdateNarrativeEstimateCalls()); n != 0 {
			t.Errorf("expected no estimate update without a newest claim, got %d", n)
		}
	})

	t.Run("UpdateEstimateError", func(t *testing.T) {
		f := newClusterFixture(nil)
		f.repo.UpdateNarrativeEstimateFunc = func(
			context.Context, uuid.UUID, *int, model.NarrativeStatus,
		) error {
			return data.ErrInternal
		}
		s := NewNarrativeService(f.repo, &narrativeStubEmbedder{}, f.generator, testMatchThreshold)

		err := s.ClusterClaim(t.Context(), f.claim.ID)
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}
	})
}

func TestNarrativeService_RecordClaims(t *testing.T) {
	newClaims := func() []model.Claim {
		return []model.Claim{
			{Text: "first claim", TopicID: uuid.New()},
			{Text: "second claim", TopicID: uuid.New()},
		}
	}

	t.Run("StoresAndEnqueuesEveryClaim", func(t *testing.T) {
		repo := newNarrativeRepoMock()
		repo.CreateClaimFunc = func(_ context.Context, c *model.Claim) (*model.Claim, error) {
			stored := *c
			stored.ID = uuid.New()
			return &stored, nil
		}
		repo.EnqueueClusterClaimFunc = func(context.Context, uuid.UUID) error { return nil }
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		err := s.RecordClaims(t.Context(), newClaims())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n := len(repo.CreateClaimCalls()); n != 2 {
			t.Fatalf("expected 2 claims stored, got %d", n)
		}
		if n := len(repo.WithTxCalls()); n != 1 {
			t.Errorf("expected all claims in one transaction, got %d", n)
		}
		enqueued := repo.EnqueueClusterClaimCalls()
		if len(enqueued) != 2 {
			t.Fatalf("expected 2 cluster jobs, got %d", len(enqueued))
		}
		if enqueued[0].ClaimID == uuid.Nil || enqueued[0].ClaimID == enqueued[1].ClaimID {
			t.Errorf("expected each job to carry its own stored claim ID, got %+v", enqueued)
		}
	})

	t.Run("SkipsClaimThatAlreadyExists", func(t *testing.T) {
		repo := newNarrativeRepoMock()
		repo.CreateClaimFunc = func(_ context.Context, c *model.Claim) (*model.Claim, error) {
			if c.Text == "first claim" {
				return nil, data.ErrConflict
			}
			stored := *c
			stored.ID = uuid.New()
			return &stored, nil
		}
		repo.EnqueueClusterClaimFunc = func(context.Context, uuid.UUID) error { return nil }
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		err := s.RecordClaims(t.Context(), newClaims())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n := len(repo.EnqueueClusterClaimCalls()); n != 1 {
			t.Errorf("expected only the new claim to be enqueued, got %d job(s)", n)
		}
	})

	t.Run("CreateClaimError", func(t *testing.T) {
		repo := newNarrativeRepoMock()
		repo.CreateClaimFunc = func(context.Context, *model.Claim) (*model.Claim, error) {
			return nil, data.ErrInternal
		}
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		err := s.RecordClaims(t.Context(), newClaims())
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}
		if n := len(repo.CreateClaimCalls()); n != 1 {
			t.Errorf("expected the loop to stop at the first failure, got %d call(s)", n)
		}
	})

	t.Run("EnqueueError", func(t *testing.T) {
		repo := newNarrativeRepoMock()
		repo.CreateClaimFunc = func(_ context.Context, c *model.Claim) (*model.Claim, error) {
			stored := *c
			stored.ID = uuid.New()
			return &stored, nil
		}
		repo.EnqueueClusterClaimFunc = func(context.Context, uuid.UUID) error {
			return data.ErrQueueFull
		}
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		err := s.RecordClaims(t.Context(), newClaims())
		if !errors.Is(err, data.ErrQueueFull) {
			t.Fatalf("expected %v, got %v", data.ErrQueueFull, err)
		}
	})

	t.Run("NoClaims", func(t *testing.T) {
		repo := newNarrativeRepoMock()
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		err := s.RecordClaims(t.Context(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n := len(repo.CreateClaimCalls()); n != 0 {
			t.Errorf("expected nothing to be stored, got %d", n)
		}
	})
}

func TestNarrativeService_ActiveTopics(t *testing.T) {
	t.Run("MapsTopics", func(t *testing.T) {
		topic := model.Topic{ID: uuid.New(), Name: "Family planning", Description: "Contraception"}
		repo := newNarrativeRepoMock()
		repo.ListActiveTopicsFunc = func(context.Context) ([]model.Topic, error) {
			return []model.Topic{topic}, nil
		}
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		got, err := s.ActiveTopics(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := data.AnalyzerTopic{ID: topic.ID, Name: topic.Name, Description: topic.Description}
		if len(got) != 1 || got[0] != want {
			t.Errorf("expected [%+v], got %+v", want, got)
		}
	})

	t.Run("RepoError", func(t *testing.T) {
		repo := newNarrativeRepoMock()
		repo.ListActiveTopicsFunc = func(context.Context) ([]model.Topic, error) {
			return nil, data.ErrInternal
		}
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		got, err := s.ActiveTopics(t.Context())
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}
		if got != nil {
			t.Errorf("expected no topics on error, got %+v", got)
		}
	})
}

func TestNarrativeService_DeactivateStale(t *testing.T) {
	const window = 24 * time.Hour

	t.Run("PassesWindowStartToRepo", func(t *testing.T) {
		repo := newNarrativeRepoMock()
		repo.DeactivateStaleNarrativesFunc = func(context.Context, time.Time) (int64, error) {
			return 4, nil
		}
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		before := time.Now()
		count, err := s.DeactivateStale(t.Context(), window)
		after := time.Now()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 4 {
			t.Errorf("expected 4 deactivated narratives, got %d", count)
		}
		calls := repo.DeactivateStaleNarrativesCalls()
		if len(calls) != 1 {
			t.Fatalf("expected 1 repo call, got %d", len(calls))
		}
		if calls[0].Since.Before(before.Add(-window)) || calls[0].Since.After(after.Add(-window)) {
			t.Errorf("expected since to be one window before now, got %v", calls[0].Since)
		}
	})

	t.Run("RepoError", func(t *testing.T) {
		repo := newNarrativeRepoMock()
		repo.DeactivateStaleNarrativesFunc = func(context.Context, time.Time) (int64, error) {
			return 0, data.ErrInternal
		}
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		_, err := s.DeactivateStale(t.Context(), window)
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}
	})
}

func TestNarrativeService_List(t *testing.T) {
	t.Run("ReturnsRepoPage", func(t *testing.T) {
		page := &dto.PaginationResults[model.Narrative]{
			Page:       2,
			PerPage:    5,
			TotalItems: 6,
			Items:      []model.Narrative{{ID: uuid.New(), Statement: "a narrative"}},
		}
		var captured data.SupportsPagination
		repo := newNarrativeRepoMock()
		repo.PaginateNarrativesFunc = func(
			_ context.Context,
			q data.SupportsPagination,
			_ data.NarrativeFilter,
		) (*dto.PaginationResults[model.Narrative], error) {
			captured = q
			return page, nil
		}
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		got, err := s.List(
			t.Context(),
			&dto.NarrativeListDTO{PaginationDTO: dto.PaginationDTO{Page: 2, PerPage: 5}},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != page {
			t.Errorf("expected the repo page to be returned as is")
		}
		if captured == nil || captured.Limit() != 5 || captured.Offset() != 5 {
			t.Errorf("expected limit 5 and offset 5 to reach the repo, got %v", captured)
		}
	})

	t.Run("RepoError", func(t *testing.T) {
		repo := newNarrativeRepoMock()
		repo.PaginateNarrativesFunc = func(
			context.Context,
			data.SupportsPagination,
			data.NarrativeFilter,
		) (*dto.PaginationResults[model.Narrative], error) {
			return nil, data.ErrInternal
		}
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		_, err := s.List(t.Context(), &dto.NarrativeListDTO{})
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}
	})
}

func TestNarrativeService_GetByID(t *testing.T) {
	t.Run("ReturnsNarrative", func(t *testing.T) {
		id := uuid.New()
		repo := newNarrativeRepoMock()
		repo.GetNarrativeFunc = func(_ context.Context, got uuid.UUID) (*model.Narrative, error) {
			return &model.Narrative{ID: got, Statement: "a narrative"}, nil
		}
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		narrative, err := s.GetByID(t.Context(), id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if narrative.ID != id {
			t.Errorf("expected narrative %s, got %s", id, narrative.ID)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		repo := newNarrativeRepoMock()
		repo.GetNarrativeFunc = func(context.Context, uuid.UUID) (*model.Narrative, error) {
			return nil, data.ErrNotFound
		}
		s := NewNarrativeService(
			repo,
			&narrativeStubEmbedder{},
			&stubStatementGenerator{},
			testMatchThreshold,
		)

		_, err := s.GetByID(t.Context(), uuid.New())
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("expected %v, got %v", data.ErrNotFound, err)
		}
	})
}

func TestEstimateHarm(t *testing.T) {
	cases := []struct {
		name     string
		severity model.Severity
		highRisk bool
		want     int
	}{
		{"RoutineIsMinimum", model.SeverityRoutine, false, 1},
		{"SeriousIsThree", model.SeveritySerious, false, 3},
		{"EmergencyIsMaximum", model.SeverityEmergency, false, 5},
		{"HighRiskLiftsRoutineToFour", model.SeverityRoutine, true, 4},
		{"HighRiskLiftsSeriousToFour", model.SeveritySerious, true, 4},
		{"HighRiskKeepsEmergencyAtFive", model.SeverityEmergency, true, 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := estimateHarm(&model.Claim{Severity: tc.severity, HighRisk: tc.highRisk})

			if got == nil || *got != tc.want {
				t.Errorf("expected harm %d, got %v", tc.want, got)
			}
		})
	}
}

func TestNarrativeStatusFor(t *testing.T) {
	cases := []struct {
		name    string
		verdict model.Verdict
		want    model.NarrativeStatus
	}{
		{
			"InsufficientEvidenceIsNotInKB",
			model.VerdictInsufficientEvidence,
			model.NarrativeStatusNotInKB,
		},
		{"SupportedIsAutoChecked", model.VerdictSupported, model.NarrativeStatusAutoChecked},
		{"ContradictedIsAutoChecked", model.VerdictContradicted, model.NarrativeStatusAutoChecked},
		{"MixedIsAutoChecked", model.VerdictMixed, model.NarrativeStatusAutoChecked},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := narrativeStatusFor(&model.Claim{Verdict: tc.verdict})

			if got != tc.want {
				t.Errorf("expected status %q, got %q", tc.want, got)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	cases := []struct {
		name string
		a, b []float32
		want float64
	}{
		{"Identical", []float32{1, 2, 3}, []float32{1, 2, 3}, 1},
		{"Orthogonal", []float32{1, 0}, []float32{0, 1}, 0},
		{"Opposite", []float32{1, 0}, []float32{-1, 0}, -1},
		{"DifferentLengths", []float32{1, 2}, []float32{1, 2, 3}, 0},
		{"ZeroVector", []float32{0, 0}, []float32{1, 2}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cosineSimilarity(tc.a, tc.b)

			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
