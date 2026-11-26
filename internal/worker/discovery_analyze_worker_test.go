package worker

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func buildAnalyzeWorker(
	store *pipelineWorkerStore,
	checker *pipelineWorkerChecker,
) *AnalyzeSingleVideoWorker {
	return NewAnalyzeSingleVideoWorker(service.NewAnalyzeService(store, checker))
}

func runAnalyzeJobAttempt(
	ctx context.Context,
	w *AnalyzeSingleVideoWorker,
	videoID string,
	attempt int,
) error {
	return w.Work(ctx, &river.Job[queue.AnalyzeSingleVideoJob]{
		JobRow: &rivertype.JobRow{Attempt: attempt, MaxAttempts: queue.AnalyzeMaxAttempts},
		Args:   queue.AnalyzeSingleVideoJob{VideoID: videoID},
	})
}

func transcribedFixture() *model.Video {
	transcript := "lemon water cures infections"
	return &model.Video{TranscriptionText: &transcript}
}

func TestAnalyzeSingleVideoWorker_Timeout(t *testing.T) {
	w := buildAnalyzeWorker(
		newPipelineWorkerStore(model.VideoStatusTranscribed),
		&pipelineWorkerChecker{},
	)

	if got := w.Timeout(&river.Job[queue.AnalyzeSingleVideoJob]{}); got != analyzeJobTimeout {
		t.Errorf("Timeout() = %v, want %v", got, analyzeJobTimeout)
	}
}

func TestAnalyzeSingleVideoWorker_Work(t *testing.T) {
	outage := errors.New("llm unavailable")

	t.Run("AnalyzesTheVideo", func(t *testing.T) {
		video := transcribedFixture()
		store := newPipelineWorkerStore(model.VideoStatusTranscribed, video)
		w := buildAnalyzeWorker(store, &pipelineWorkerChecker{inScope: true})

		if err := runAnalyzeJobAttempt(t.Context(), w, video.ID.String(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := store.status[video.ID]; got != model.VideoStatusVerified {
			t.Errorf("status = %q, want verified", got)
		}
	})

	t.Run("OutOfScopeVideo", func(t *testing.T) {
		video := transcribedFixture()
		store := newPipelineWorkerStore(model.VideoStatusTranscribed, video)
		w := buildAnalyzeWorker(store, &pipelineWorkerChecker{inScope: false})

		if err := runAnalyzeJobAttempt(t.Context(), w, video.ID.String(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := store.status[video.ID]; got != model.VideoStatusOutOfScope {
			t.Errorf("status = %q, want out_of_scope", got)
		}
	})

	t.Run("InvalidVideoID_CancelsTheJob", func(t *testing.T) {
		w := buildAnalyzeWorker(
			newPipelineWorkerStore(model.VideoStatusTranscribed),
			&pipelineWorkerChecker{},
		)

		err := runAnalyzeJobAttempt(t.Context(), w, "not-a-uuid", 1)

		var cancelErr *rivertype.JobCancelError
		if !errors.As(err, &cancelErr) {
			t.Fatalf("error = %v, want a job cancel error", err)
		}
	})

	t.Run("FailureBeforeLastAttempt_KeepsVideoTranscribed", func(t *testing.T) {
		video := transcribedFixture()
		store := newPipelineWorkerStore(model.VideoStatusTranscribed, video)
		w := buildAnalyzeWorker(store, &pipelineWorkerChecker{err: outage})

		err := runAnalyzeJobAttempt(
			t.Context(), w, video.ID.String(), queue.AnalyzeMaxAttempts-1,
		)

		if !errors.Is(err, outage) {
			t.Fatalf("error = %v, want it to wrap the checker error", err)
		}
		var cancelErr *rivertype.JobCancelError
		if errors.As(err, &cancelErr) {
			t.Error("an outage must fail the job normally, not cancel it")
		}
		if got := store.status[video.ID]; got != model.VideoStatusTranscribed {
			t.Errorf("status = %q, want transcribed", got)
		}
	})

	t.Run("FailureOnLastAttempt_MarksVideoFailed", func(t *testing.T) {
		video := transcribedFixture()
		store := newPipelineWorkerStore(model.VideoStatusTranscribed, video)
		w := buildAnalyzeWorker(store, &pipelineWorkerChecker{err: outage})

		err := runAnalyzeJobAttempt(t.Context(), w, video.ID.String(), queue.AnalyzeMaxAttempts)

		if !errors.Is(err, outage) {
			t.Fatalf("error = %v, want it to wrap the checker error", err)
		}
		if got := store.status[video.ID]; got != model.VideoStatusFailed {
			t.Errorf("status = %q, want failed", got)
		}
		if got := store.failedStage[video.ID]; got != model.VideoFailedStageAnalyze {
			t.Errorf("failed stage = %q, want %q", got, model.VideoFailedStageAnalyze)
		}
	})

	t.Run("FailureOnLastAttemptDuringShutdown_KeepsVideoTranscribed", func(t *testing.T) {
		video := transcribedFixture()
		store := newPipelineWorkerStore(model.VideoStatusTranscribed, video)
		w := buildAnalyzeWorker(store, &pipelineWorkerChecker{err: context.Canceled})
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := runAnalyzeJobAttempt(ctx, w, video.ID.String(), queue.AnalyzeMaxAttempts)

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want it to wrap context.Canceled", err)
		}
		if got := store.status[video.ID]; got != model.VideoStatusTranscribed {
			t.Errorf("status = %q, a shutdown must not mark the video as failed", got)
		}
	})

	t.Run("MarkFailedError_IsJoinedToTheFailure", func(t *testing.T) {
		boom := errors.New("db down")
		video := transcribedFixture()
		store := newPipelineWorkerStore(model.VideoStatusTranscribed, video)
		store.failErr = boom
		w := buildAnalyzeWorker(store, &pipelineWorkerChecker{err: outage})

		err := runAnalyzeJobAttempt(t.Context(), w, video.ID.String(), queue.AnalyzeMaxAttempts)

		if !errors.Is(err, outage) || !errors.Is(err, boom) {
			t.Fatalf("error = %v, want both the checker error and the mark error", err)
		}
	})
}

func TestAnalyzeSingleVideoWorker_NextRetry(t *testing.T) {
	w := buildAnalyzeWorker(
		newPipelineWorkerStore(model.VideoStatusTranscribed),
		&pipelineWorkerChecker{},
	)

	tests := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{"first failure", 1, 5 * time.Minute},
		{"second failure", 2, 15 * time.Minute},
		{"beyond the delay table", 3, 15 * time.Minute},
		{"attempt zero is treated as the first", 0, 5 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &river.Job[queue.AnalyzeSingleVideoJob]{
				JobRow: &rivertype.JobRow{Attempt: tt.attempt},
			}

			got := time.Until(w.NextRetry(job))

			if got > tt.want || got < tt.want-time.Minute {
				t.Errorf("retry in %v, want about %v", got, tt.want)
			}
		})
	}
}

func buildAnalyzeMultipleVideoWorker(store *pipelineWorkerStore) *AnalyzeMultipleVideoWorker {
	return NewAnalyzeMultipleVideoWorker(service.NewAnalyzeService(store, &pipelineWorkerChecker{}))
}

func runAnalyzeMultipleVideoJob(t *testing.T, w *AnalyzeMultipleVideoWorker) error {
	t.Helper()
	return w.Work(t.Context(), &river.Job[queue.AnalyzeMultipleVideoJob]{})
}

func TestAnalyzeMultipleVideoWorker_Timeout(t *testing.T) {
	w := buildAnalyzeMultipleVideoWorker(newPipelineWorkerStore(model.VideoStatusTranscribed))

	if got := w.Timeout(
		&river.Job[queue.AnalyzeMultipleVideoJob]{},
	); got != analyzeDispatchTimeout {
		t.Errorf("Timeout() = %v, want %v", got, analyzeDispatchTimeout)
	}
}

func TestAnalyzeMultipleVideoWorker_Work(t *testing.T) {
	t.Run("EnqueuesEveryTranscribedVideo", func(t *testing.T) {
		a, b, other := transcribedFixture(), transcribedFixture(), transcribedFixture()
		store := newPipelineWorkerStore(model.VideoStatusTranscribed, a, b, other)
		store.status[other.ID] = model.VideoStatusVerified

		if err := runAnalyzeMultipleVideoJob(
			t,
			buildAnalyzeMultipleVideoWorker(store),
		); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if want := []uuid.UUID{a.ID, b.ID}; !slices.Equal(store.analyzeEnqueued, want) {
			t.Errorf("enqueued = %v, want %v", store.analyzeEnqueued, want)
		}
		if store.status[a.ID] != model.VideoStatusTranscribed {
			t.Error("the dispatch run must not analyze videos itself")
		}
	})

	t.Run("NothingToEnqueue_Succeeds", func(t *testing.T) {
		w := buildAnalyzeMultipleVideoWorker(newPipelineWorkerStore(model.VideoStatusTranscribed))

		if err := runAnalyzeMultipleVideoJob(t, w); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ListFailure_ReturnsError", func(t *testing.T) {
		boom := errors.New("db down")
		store := newPipelineWorkerStore(model.VideoStatusTranscribed)
		store.listErr = boom

		err := runAnalyzeMultipleVideoJob(t, buildAnalyzeMultipleVideoWorker(store))

		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want it to wrap the list error", err)
		}
	})

	t.Run("EnqueueFailure_ReturnsError", func(t *testing.T) {
		boom := errors.New("queue down")
		store := newPipelineWorkerStore(model.VideoStatusTranscribed, transcribedFixture())
		store.enqueueErr = boom

		err := runAnalyzeMultipleVideoJob(t, buildAnalyzeMultipleVideoWorker(store))

		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want it to wrap the enqueue error", err)
		}
	})
}
