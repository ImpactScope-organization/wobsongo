package worker

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type pipelineWorkerStore struct {
	data.VideoRepoer
	videos          map[uuid.UUID]*model.Video
	status          map[uuid.UUID]model.VideoStatus
	failedStage     map[uuid.UUID]string
	analyzeEnqueued []uuid.UUID
	failErr         error
}

func newPipelineWorkerStore(status model.VideoStatus, videos ...*model.Video) *pipelineWorkerStore {
	s := &pipelineWorkerStore{
		videos:      make(map[uuid.UUID]*model.Video),
		status:      make(map[uuid.UUID]model.VideoStatus),
		failedStage: make(map[uuid.UUID]string),
	}
	for _, v := range videos {
		v.ID = uuid.New()
		s.videos[v.ID] = v
		s.status[v.ID] = status
	}
	return s
}

func (s *pipelineWorkerStore) WithTx(_ context.Context, fn func(data.VideoRepoer) error) error {
	return fn(s)
}

func (s *pipelineWorkerStore) GetVideoForTranscription(
	_ context.Context,
	id uuid.UUID,
) (*model.Video, error) {
	if s.status[id] != model.VideoStatusScreenedIn {
		return nil, data.ErrNotFound
	}
	return s.videos[id], nil
}

func (s *pipelineWorkerStore) SetVideoMediaURL(_ context.Context, id uuid.UUID, url string) error {
	s.videos[id].MediaURL = &url
	return nil
}

func (s *pipelineWorkerStore) SetVideoTranscribed(
	_ context.Context,
	id uuid.UUID,
	_ string,
) (bool, error) {
	if s.status[id] != model.VideoStatusScreenedIn {
		return false, nil
	}
	s.status[id] = model.VideoStatusTranscribed
	return true, nil
}

func (s *pipelineWorkerStore) EnqueueAnalyzeVideoJob(
	_ context.Context,
	id uuid.UUID,
) (bool, error) {
	s.analyzeEnqueued = append(s.analyzeEnqueued, id)
	return true, nil
}

func (s *pipelineWorkerStore) GetVideoForAnalysis(
	_ context.Context,
	id uuid.UUID,
) (*model.Video, error) {
	if s.status[id] != model.VideoStatusTranscribed {
		return nil, data.ErrNotFound
	}
	return s.videos[id], nil
}

func (s *pipelineWorkerStore) SetVideoAnalyzed(
	_ context.Context,
	id uuid.UUID,
	status model.VideoStatus,
) (bool, error) {
	if s.status[id] != model.VideoStatusTranscribed {
		return false, nil
	}
	s.status[id] = status
	return true, nil
}

func (s *pipelineWorkerStore) SetVideoFailed(
	_ context.Context,
	id uuid.UUID,
	fromStatus model.VideoStatus,
	stage, _ string,
) (bool, error) {
	if s.failErr != nil {
		return false, s.failErr
	}
	if s.status[id] != fromStatus {
		return false, nil
	}
	s.status[id] = model.VideoStatusFailed
	s.failedStage[id] = stage
	return true, nil
}

type pipelineWorkerResolver struct{ err error }

func (r *pipelineWorkerResolver) ResolveMediaURL(context.Context, string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return "https://media.test/video.mp4", nil
}

type pipelineWorkerTranscriber struct{ err error }

func (tr *pipelineWorkerTranscriber) TranscribeURL(
	context.Context,
	string,
	string,
) (string, error) {
	if tr.err != nil {
		return "", tr.err
	}
	return "transcript", nil
}

func buildTranscribeWorker(
	store *pipelineWorkerStore,
	resolver *pipelineWorkerResolver,
	transcriber *pipelineWorkerTranscriber,
) *TranscribeDiscoveredVideoWorker {
	return NewTranscribeDiscoveredVideoWorker(
		service.NewTranscribeService(store, resolver, transcriber),
	)
}

func runTranscribeJobAttempt(
	ctx context.Context,
	w *TranscribeDiscoveredVideoWorker,
	videoID string,
	attempt int,
) error {
	return w.Work(ctx, &river.Job[queue.TranscribeDiscoveredVideoJob]{
		JobRow: &rivertype.JobRow{Attempt: attempt, MaxAttempts: queue.TranscribeMaxAttempts},
		Args:   queue.TranscribeDiscoveredVideoJob{VideoID: videoID},
	})
}

func TestTranscribeDiscoveredVideoWorker_Timeout(t *testing.T) {
	w := buildTranscribeWorker(
		newPipelineWorkerStore(model.VideoStatusScreenedIn),
		&pipelineWorkerResolver{},
		&pipelineWorkerTranscriber{},
	)

	got := w.Timeout(&river.Job[queue.TranscribeDiscoveredVideoJob]{})
	if got != transcribeJobTimeout {
		t.Errorf("Timeout() = %v, want %v", got, transcribeJobTimeout)
	}
}

func TestTranscribeDiscoveredVideoWorker_Work(t *testing.T) {
	outage := errors.New("asr unavailable")

	t.Run("TranscribesTheVideo", func(t *testing.T) {
		video := &model.Video{VideoURL: "https://www.tiktok.com/@a/video/1"}
		store := newPipelineWorkerStore(model.VideoStatusScreenedIn, video)
		w := buildTranscribeWorker(store, &pipelineWorkerResolver{}, &pipelineWorkerTranscriber{})

		if err := runTranscribeJobAttempt(t.Context(), w, video.ID.String(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := store.status[video.ID]; got != model.VideoStatusTranscribed {
			t.Errorf("status = %q, want transcribed", got)
		}
		if !slices.Equal(store.analyzeEnqueued, []uuid.UUID{video.ID}) {
			t.Errorf("analysis queued for %v, want %v", store.analyzeEnqueued, video.ID)
		}
	})

	t.Run("InvalidVideoID_CancelsTheJob", func(t *testing.T) {
		w := buildTranscribeWorker(
			newPipelineWorkerStore(model.VideoStatusScreenedIn),
			&pipelineWorkerResolver{},
			&pipelineWorkerTranscriber{},
		)

		err := runTranscribeJobAttempt(t.Context(), w, "not-a-uuid", 1)

		var cancelErr *rivertype.JobCancelError
		if !errors.As(err, &cancelErr) {
			t.Fatalf("error = %v, want a job cancel error", err)
		}
	})

	t.Run("VideoThatMovedOn_Succeeds", func(t *testing.T) {
		video := &model.Video{}
		store := newPipelineWorkerStore(model.VideoStatusTranscribed, video)
		w := buildTranscribeWorker(store, &pipelineWorkerResolver{}, &pipelineWorkerTranscriber{})

		if err := runTranscribeJobAttempt(t.Context(), w, video.ID.String(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(store.analyzeEnqueued) != 0 {
			t.Error("a video that moved on must not queue another analysis")
		}
	})

	t.Run("FailureBeforeLastAttempt_KeepsVideoScreenedIn", func(t *testing.T) {
		video := &model.Video{}
		store := newPipelineWorkerStore(model.VideoStatusScreenedIn, video)
		w := buildTranscribeWorker(
			store,
			&pipelineWorkerResolver{},
			&pipelineWorkerTranscriber{err: outage},
		)

		err := runTranscribeJobAttempt(
			t.Context(), w, video.ID.String(), queue.TranscribeMaxAttempts-1,
		)

		if !errors.Is(err, outage) {
			t.Fatalf("error = %v, want it to wrap the asr error", err)
		}
		var cancelErr *rivertype.JobCancelError
		if errors.As(err, &cancelErr) {
			t.Error("an outage must fail the job normally, not cancel it")
		}
		if got := store.status[video.ID]; got != model.VideoStatusScreenedIn {
			t.Errorf("status = %q, want screened_in", got)
		}
	})

	t.Run("FailureOnLastAttempt_MarksVideoFailed", func(t *testing.T) {
		video := &model.Video{}
		store := newPipelineWorkerStore(model.VideoStatusScreenedIn, video)
		w := buildTranscribeWorker(
			store,
			&pipelineWorkerResolver{},
			&pipelineWorkerTranscriber{err: outage},
		)

		err := runTranscribeJobAttempt(
			t.Context(), w, video.ID.String(), queue.TranscribeMaxAttempts,
		)

		if !errors.Is(err, outage) {
			t.Fatalf("error = %v, want it to wrap the asr error", err)
		}
		if got := store.status[video.ID]; got != model.VideoStatusFailed {
			t.Errorf("status = %q, want failed", got)
		}
		if got := store.failedStage[video.ID]; got != model.VideoFailedStageTranscribe {
			t.Errorf("failed stage = %q, want %q", got, model.VideoFailedStageTranscribe)
		}
	})

	t.Run("FailureOnLastAttemptDuringShutdown_KeepsVideoScreenedIn", func(t *testing.T) {
		video := &model.Video{}
		store := newPipelineWorkerStore(model.VideoStatusScreenedIn, video)
		w := buildTranscribeWorker(
			store, &pipelineWorkerResolver{}, &pipelineWorkerTranscriber{err: context.Canceled},
		)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := runTranscribeJobAttempt(ctx, w, video.ID.String(), queue.TranscribeMaxAttempts)

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want it to wrap context.Canceled", err)
		}
		if got := store.status[video.ID]; got != model.VideoStatusScreenedIn {
			t.Errorf("status = %q, a shutdown must not mark the video as failed", got)
		}
	})

	t.Run("MediaUnavailable_MarksFailedAndCancelsWithoutRetry", func(t *testing.T) {
		video := &model.Video{}
		store := newPipelineWorkerStore(model.VideoStatusScreenedIn, video)
		w := buildTranscribeWorker(
			store,
			&pipelineWorkerResolver{err: data.ErrMediaUnavailable},
			&pipelineWorkerTranscriber{},
		)

		// First attempt: retrying would only spend another Apify run.
		err := runTranscribeJobAttempt(t.Context(), w, video.ID.String(), 1)

		var cancelErr *rivertype.JobCancelError
		if !errors.As(err, &cancelErr) {
			t.Fatalf("error = %v, want a job cancel error", err)
		}
		if got := store.status[video.ID]; got != model.VideoStatusFailed {
			t.Errorf("status = %q, want failed", got)
		}
	})

	t.Run("MarkFailedError_IsJoinedToTheFailure", func(t *testing.T) {
		boom := errors.New("db down")
		video := &model.Video{}
		store := newPipelineWorkerStore(model.VideoStatusScreenedIn, video)
		store.failErr = boom
		w := buildTranscribeWorker(
			store,
			&pipelineWorkerResolver{},
			&pipelineWorkerTranscriber{err: outage},
		)

		err := runTranscribeJobAttempt(
			t.Context(), w, video.ID.String(), queue.TranscribeMaxAttempts,
		)

		if !errors.Is(err, outage) || !errors.Is(err, boom) {
			t.Fatalf("error = %v, want both the asr error and the mark error", err)
		}
	})
}

func TestTranscribeDiscoveredVideoWorker_NextRetry(t *testing.T) {
	w := buildTranscribeWorker(
		newPipelineWorkerStore(model.VideoStatusScreenedIn),
		&pipelineWorkerResolver{},
		&pipelineWorkerTranscriber{},
	)

	tests := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{"first failure", 1, 2 * time.Minute},
		{"second failure", 2, 10 * time.Minute},
		{"beyond the delay table", 3, 10 * time.Minute},
		{"attempt zero is treated as the first", 0, 2 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &river.Job[queue.TranscribeDiscoveredVideoJob]{
				JobRow: &rivertype.JobRow{Attempt: tt.attempt},
			}

			got := time.Until(w.NextRetry(job))

			if got > tt.want || got < tt.want-time.Minute {
				t.Errorf("retry in %v, want about %v", got, tt.want)
			}
		})
	}
}

// pipelineWorkerChecker is a service.ClaimChecker.
type pipelineWorkerChecker struct {
	inScope bool
	err     error
}

func (c *pipelineWorkerChecker) CheckClaim(
	context.Context,
	*dto.CheckClaimDTO,
) (*service.ClaimCheckResult, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &service.ClaimCheckResult{InScope: c.inScope}, nil
}
