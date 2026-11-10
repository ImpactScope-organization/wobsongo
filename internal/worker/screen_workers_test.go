package worker

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// screenWorkerStore is an in-memory store with the two methods
// service.ScreenService needs from a video repo.
type screenWorkerStore struct {
	mu      sync.Mutex
	videos  []*model.Video
	status  map[uuid.UUID]model.VideoStatus
	listErr error
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
	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *screenWorkerStore) SetVideoScreenStatus(
	_ context.Context,
	id uuid.UUID,
	status model.VideoStatus,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, screened := s.status[id]; screened {
		return false, nil
	}
	s.status[id] = status
	return true, nil
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

func newScreenWorker(store *screenWorkerStore, analyzer *screenWorkerAnalyzer) *ScreenVideosWorker {
	return NewScreenVideosWorker(service.NewScreenService(store, analyzer, 2))
}

func runScreenJob(t *testing.T, w *ScreenVideosWorker) error {
	t.Helper()
	return w.Work(t.Context(), &river.Job[queue.ScreenVideosJob]{})
}

func TestScreenVideosWorker_Timeout(t *testing.T) {
	w := newScreenWorker(newScreenWorkerStore(), &screenWorkerAnalyzer{})

	if got := w.Timeout(&river.Job[queue.ScreenVideosJob]{}); got != screenJobTimeout {
		t.Errorf("Timeout() = %v, want %v", got, screenJobTimeout)
	}
}

func TestScreenVideosWorker_Work(t *testing.T) {
	t.Run("ScreensEveryDiscoveredVideo", func(t *testing.T) {
		in := &model.Video{Caption: "pilule contraceptive", Hashtags: []string{"santé"}}
		out := &model.Video{Caption: "recette de crêpes"}
		store := newScreenWorkerStore(in, out)
		analyzer := &screenWorkerAnalyzer{
			inScope: map[string]bool{"pilule contraceptive #santé": true},
		}

		if err := runScreenJob(t, newScreenWorker(store, analyzer)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := store.status[in.ID]; got != model.VideoStatusScreenedIn {
			t.Errorf("in-scope video status = %q, want screened_in", got)
		}
		if got := store.status[out.ID]; got != model.VideoStatusScreenedOut {
			t.Errorf("out-of-scope video status = %q, want screened_out", got)
		}
	})

	t.Run("NothingToScreen_Succeeds", func(t *testing.T) {
		w := newScreenWorker(newScreenWorkerStore(), &screenWorkerAnalyzer{})

		if err := runScreenJob(t, w); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("AnalyzerOutage_ReturnsErrorWithoutCancel", func(t *testing.T) {
		outage := errors.New("llm unavailable")
		video := &model.Video{Caption: "claim"}
		store := newScreenWorkerStore(video)

		err := runScreenJob(t, newScreenWorker(store, &screenWorkerAnalyzer{err: outage}))

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

	t.Run("ListFailure_ReturnsError", func(t *testing.T) {
		boom := errors.New("db down")
		store := newScreenWorkerStore()
		store.listErr = boom

		err := runScreenJob(t, newScreenWorker(store, &screenWorkerAnalyzer{}))

		if !errors.Is(err, boom) {
			t.Fatalf("error = %v, want it to wrap the list error", err)
		}
	})
}
