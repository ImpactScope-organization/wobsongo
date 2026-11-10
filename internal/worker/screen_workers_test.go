package worker

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// screenWorkerStore is an in-memory store with the methods
// service.ScreenService needs from a video repo.
type screenWorkerStore struct {
	videos     []*model.Video
	status     map[uuid.UUID]model.VideoStatus
	enqueued   []uuid.UUID
	listErr    error
	enqueueErr error
}

func newScreenWorkerStore(videos ...*model.Video) *screenWorkerStore {
	for _, v := range videos {
		v.ID = uuid.New()
	}
	return &screenWorkerStore{
		videos: videos,
		status: make(map[uuid.UUID]model.VideoStatus),
	}
}

func (s *screenWorkerStore) ListDiscoveredVideos(
	_ context.Context,
	limit int32,
	excludeIDs []uuid.UUID,
) ([]*model.Video, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}

	var out []*model.Video
	for _, v := range s.videos {
		if int32(len(out)) >= limit {
			break
		}
		if _, screened := s.status[v.ID]; screened || slices.Contains(excludeIDs, v.ID) {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *screenWorkerStore) GetVideoForScreening(
	_ context.Context,
	id uuid.UUID,
) (*model.Video, error) {
	if _, screened := s.status[id]; !screened {
		for _, v := range s.videos {
			if v.ID == id {
				return v, nil
			}
		}
	}
	return nil, data.ErrNotFound
}

func (s *screenWorkerStore) SetVideoScreenStatus(
	_ context.Context,
	id uuid.UUID,
	status model.VideoStatus,
) (bool, error) {
	if _, screened := s.status[id]; screened {
		return false, nil
	}
	s.status[id] = status
	return true, nil
}

func (s *screenWorkerStore) EnqueueScreenVideoJob(_ context.Context, videoID uuid.UUID) error {
	if s.enqueueErr != nil {
		return s.enqueueErr
	}
	s.enqueued = append(s.enqueued, videoID)
	return nil
}

// screenWorkerAnalyzer is a hand-rolled data.ClaimAnalyzer.
type screenWorkerAnalyzer struct {
	inScope map[string]bool
	err     error
}

func (a *screenWorkerAnalyzer) Analyze(
	_ context.Context,
	message string,
	_ []data.AnalyzerTopic,
) (*data.ClaimAnalysis, error) {
	if a.err != nil {
		return nil, a.err
	}
	return &data.ClaimAnalysis{InScope: a.inScope[message]}, nil
}

func buildScreenVideosWorker(
	store *screenWorkerStore,
	analyzer *screenWorkerAnalyzer,
) *ScreenVideosWorker {
	return NewScreenVideosWorker(service.NewScreenService(store, analyzer))
}

func buildScreenVideoWorker(
	store *screenWorkerStore,
	analyzer *screenWorkerAnalyzer,
) *ScreenVideoWorker {
	return NewScreenVideoWorker(service.NewScreenService(store, analyzer))
}

func runScreenVideosJob(t *testing.T, w *ScreenVideosWorker) error {
	t.Helper()
	return w.Work(t.Context(), &river.Job[queue.ScreenVideosJob]{})
}

func runScreenVideoJob(t *testing.T, w *ScreenVideoWorker, videoID string) error {
	t.Helper()
	return w.Work(t.Context(), &river.Job[queue.ScreenVideoJob]{
		Args: queue.ScreenVideoJob{VideoID: videoID},
	})
}

func TestScreenVideosWorker_Timeout(t *testing.T) {
	w := buildScreenVideosWorker(newScreenWorkerStore(), &screenWorkerAnalyzer{})

	if got := w.Timeout(&river.Job[queue.ScreenVideosJob]{}); got != screenDispatchTimeout {
		t.Errorf("Timeout() = %v, want %v", got, screenDispatchTimeout)
	}
}

func TestScreenVideosWorker_Work(t *testing.T) {
	t.Run("EnqueuesEveryDiscoveredVideo", func(t *testing.T) {
		a := &model.Video{Caption: "pilule contraceptive"}
		b := &model.Video{Caption: "recette de crêpes"}
		store := newScreenWorkerStore(a, b)

		err := runScreenVideosJob(t, buildScreenVideosWorker(store, &screenWorkerAnalyzer{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if want := []uuid.UUID{a.ID, b.ID}; !slices.Equal(store.enqueued, want) {
			t.Errorf("enqueued = %v, want %v", store.enqueued, want)
		}
		if len(store.status) != 0 {
			t.Error("the dispatch run must not screen videos itself")
		}
	})

	t.Run("NothingToEnqueue_Succeeds", func(t *testing.T) {
		w := buildScreenVideosWorker(newScreenWorkerStore(), &screenWorkerAnalyzer{})

		if err := runScreenVideosJob(t, w); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ListFailure_ReturnsError", func(t *testing.T) {
		boom := errors.New("db down")
		store := newScreenWorkerStore()
		store.listErr = boom

		err := runScreenVideosJob(t, buildScreenVideosWorker(store, &screenWorkerAnalyzer{}))

		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want it to wrap the list error", err)
		}
	})

	t.Run("EnqueueFailure_ReturnsError", func(t *testing.T) {
		boom := errors.New("queue down")
		store := newScreenWorkerStore(&model.Video{Caption: "claim"})
		store.enqueueErr = boom

		err := runScreenVideosJob(t, buildScreenVideosWorker(store, &screenWorkerAnalyzer{}))

		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want it to wrap the enqueue error", err)
		}
	})
}

func TestScreenVideoWorker_Timeout(t *testing.T) {
	w := buildScreenVideoWorker(newScreenWorkerStore(), &screenWorkerAnalyzer{})

	if got := w.Timeout(&river.Job[queue.ScreenVideoJob]{}); got != screenVideoJobTimeout {
		t.Errorf("Timeout() = %v, want %v", got, screenVideoJobTimeout)
	}
}

func TestScreenVideoWorker_Work(t *testing.T) {
	t.Run("ScreensTheVideo", func(t *testing.T) {
		in := &model.Video{Caption: "pilule contraceptive", Hashtags: []string{"santé"}}
		out := &model.Video{Caption: "recette de crêpes"}
		store := newScreenWorkerStore(in, out)
		analyzer := &screenWorkerAnalyzer{
			inScope: map[string]bool{"pilule contraceptive #santé": true},
		}
		w := buildScreenVideoWorker(store, analyzer)

		if err := runScreenVideoJob(t, w, in.ID.String()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := runScreenVideoJob(t, w, out.ID.String()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := store.status[in.ID]; got != model.VideoStatusScreenedIn {
			t.Errorf("in-scope video status = %q, want screened_in", got)
		}
		if got := store.status[out.ID]; got != model.VideoStatusScreenedOut {
			t.Errorf("out-of-scope video status = %q, want screened_out", got)
		}
	})

	t.Run("AnalyzerOutage_ReturnsErrorWithoutCancel", func(t *testing.T) {
		outage := errors.New("llm unavailable")
		video := &model.Video{Caption: "claim"}
		store := newScreenWorkerStore(video)
		w := buildScreenVideoWorker(store, &screenWorkerAnalyzer{err: outage})

		err := runScreenVideoJob(t, w, video.ID.String())

		if !errors.Is(err, outage) {
			t.Fatalf("error = %v, want it to wrap the analyzer error", err)
		}
		var cancelErr *rivertype.JobCancelError
		if errors.As(err, &cancelErr) {
			t.Error("an outage must fail the job normally, not cancel it")
		}
		if _, screened := store.status[video.ID]; screened {
			t.Error("video must stay discovered after an outage")
		}
	})

	t.Run("InvalidVideoID_CancelsTheJob", func(t *testing.T) {
		w := buildScreenVideoWorker(newScreenWorkerStore(), &screenWorkerAnalyzer{})

		err := runScreenVideoJob(t, w, "not-a-uuid")

		var cancelErr *rivertype.JobCancelError
		if !errors.As(err, &cancelErr) {
			t.Fatalf("error = %v, want a job cancel error", err)
		}
	})

	t.Run("AlreadyScreenedVideo_Succeeds", func(t *testing.T) {
		video := &model.Video{Caption: "claim"}
		store := newScreenWorkerStore(video)
		store.status[video.ID] = model.VideoStatusScreenedIn
		w := buildScreenVideoWorker(store, &screenWorkerAnalyzer{})

		if err := runScreenVideoJob(t, w, video.ID.String()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := store.status[video.ID]; got != model.VideoStatusScreenedIn {
			t.Errorf("status = %q, want it to stay screened_in", got)
		}
	})
}
