package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const workerTestMatchThreshold = 0.8

// stubNarrativeStatementGenerator is a hand-rolled
// data.NarrativeStatementGenerator.
type stubNarrativeStatementGenerator struct {
	statement string
	err       error
	calls     []string
}

func (s *stubNarrativeStatementGenerator) Generate(
	_ context.Context,
	claim string,
) (string, error) {
	s.calls = append(s.calls, claim)
	if s.err != nil {
		return "", s.err
	}
	return s.statement, nil
}

// newNarrativeWorkerRepo returns a mock whose WithTx runs fn against the mock
// itself, so calls made inside the transaction are recorded on it.
func newNarrativeWorkerRepo() *mockrepo.NarrativeRepoerMock {
	repo := &mockrepo.NarrativeRepoerMock{}
	repo.WithTxFunc = func(_ context.Context, fn func(data.NarrativeRepoer) error) error {
		return fn(repo)
	}
	return repo
}

func newNarrativeWorkerService(
	repo *mockrepo.NarrativeRepoerMock,
	generator data.NarrativeStatementGenerator,
) *service.NarrativeService {
	return service.NewNarrativeService(
		repo,
		&stubEmbedder{},
		generator,
		workerTestMatchThreshold,
	)
}

func newClusterClaimJob(claimID string) *river.Job[queue.ClusterClaimJob] {
	return &river.Job[queue.ClusterClaimJob]{
		Args: queue.ClusterClaimJob{ClaimID: claimID},
	}
}

func TestClusterClaimWorker_Work(t *testing.T) {
	t.Run("InvalidClaimID_CancelsWithoutRetry", func(t *testing.T) {
		repo := newNarrativeWorkerRepo()
		w := NewClusterClaimWorker(
			newNarrativeWorkerService(repo, &stubNarrativeStatementGenerator{}),
		)

		err := w.Work(t.Context(), newClusterClaimJob("not-a-uuid"))
		if err == nil {
			t.Fatal("expected an error for a malformed claim id")
		}

		var cancelErr *rivertype.JobCancelError
		if !errors.As(err, &cancelErr) {
			t.Errorf("expected a river JobCancel error, got %T: %v", err, err)
		}
		if n := len(repo.GetClaimCalls()); n != 0 {
			t.Errorf("expected the service not to be called, got %d GetClaim call(s)", n)
		}
	})

	t.Run("ServiceError_IsWrappedAndRetried", func(t *testing.T) {
		repo := newNarrativeWorkerRepo()
		repo.GetClaimFunc = func(context.Context, uuid.UUID) (*model.Claim, error) {
			return nil, data.ErrInternal
		}
		w := NewClusterClaimWorker(
			newNarrativeWorkerService(repo, &stubNarrativeStatementGenerator{}),
		)

		err := w.Work(t.Context(), newClusterClaimJob(uuid.NewString()))
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}

		var cancelErr *rivertype.JobCancelError
		if errors.As(err, &cancelErr) {
			t.Error("a service failure must be retried, not cancelled")
		}
	})

	t.Run("Success_GroupsClaimIntoNewNarrative", func(t *testing.T) {
		claim := &model.Claim{
			ID:       uuid.New(),
			TopicID:  uuid.New(),
			Text:     "lemon water cures infections",
			Severity: model.SeveritySerious,
			Verdict:  model.VerdictContradicted,
		}
		generator := &stubNarrativeStatementGenerator{statement: "Lemon water heals illnesses"}

		repo := newNarrativeWorkerRepo()
		repo.GetClaimFunc = func(context.Context, uuid.UUID) (*model.Claim, error) {
			return claim, nil
		}
		repo.LockTopicFunc = func(context.Context, uuid.UUID) error { return nil }
		repo.ListNarrativesByTopicFunc = func(
			context.Context, uuid.UUID,
		) ([]model.Narrative, error) {
			return nil, nil
		}
		repo.CreateNarrativeFunc = func(
			_ context.Context, n *model.Narrative,
		) (*model.Narrative, error) {
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

		w := NewClusterClaimWorker(newNarrativeWorkerService(repo, generator))
		if err := w.Work(t.Context(), newClusterClaimJob(claim.ID.String())); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(generator.calls) != 1 || generator.calls[0] != claim.Text {
			t.Errorf("expected one statement written from the claim, got %v", generator.calls)
		}
		links := repo.LinkClaimCalls()
		if len(links) != 1 || links[0].ClaimID != claim.ID {
			t.Errorf("expected the claim to be linked once, got %+v", links)
		}
	})
}

func TestClusterClaimWorker_Timeout(t *testing.T) {
	w := NewClusterClaimWorker(
		newNarrativeWorkerService(newNarrativeWorkerRepo(), &stubNarrativeStatementGenerator{}),
	)

	if got := w.Timeout(newClusterClaimJob(uuid.NewString())); got != clusterClaimJobTimeout {
		t.Errorf("expected timeout %s, got %s", clusterClaimJobTimeout, got)
	}
}

func TestDeactivateNarrativesWorker_Work(t *testing.T) {
	window := 72 * time.Hour
	job := &river.Job[queue.DeactivateNarrativesJob]{}

	t.Run("PassesWindowStartToRepo", func(t *testing.T) {
		var since time.Time
		repo := newNarrativeWorkerRepo()
		repo.DeactivateStaleNarrativesFunc = func(_ context.Context, s time.Time) (int64, error) {
			since = s
			return 3, nil
		}
		w := NewDeactivateNarrativesWorker(
			newNarrativeWorkerService(repo, &stubNarrativeStatementGenerator{}),
			window,
		)

		before := time.Now()
		if err := w.Work(t.Context(), job); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		after := time.Now()

		// since is "now - window", with now read somewhere between before and after.
		if since.Before(before.Add(-window)) || since.After(after.Add(-window)) {
			t.Errorf("expected since within [%s, %s], got %s",
				before.Add(-window), after.Add(-window), since)
		}
	})

	t.Run("ServiceError_IsWrapped", func(t *testing.T) {
		repo := newNarrativeWorkerRepo()
		repo.DeactivateStaleNarrativesFunc = func(context.Context, time.Time) (int64, error) {
			return 0, data.ErrInternal
		}
		w := NewDeactivateNarrativesWorker(
			newNarrativeWorkerService(repo, &stubNarrativeStatementGenerator{}),
			window,
		)

		err := w.Work(t.Context(), job)
		if !errors.Is(err, data.ErrInternal) {
			t.Fatalf("expected %v, got %v", data.ErrInternal, err)
		}
	})
}
