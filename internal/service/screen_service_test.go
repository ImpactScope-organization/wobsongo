package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

var errScreenBoom = errors.New("llm unavailable")

// fakeScreenStore is an in-memory videoScreenStore. A video is listed until
// its status is set.
type fakeScreenStore struct {
	mu          sync.Mutex
	order       []uuid.UUID
	videos      map[uuid.UUID]*model.Video
	pending     map[uuid.UUID]bool
	status      map[uuid.UUID]model.VideoStatus
	listErr     error
	setErrs     map[uuid.UUID]error
	gotLimit    int32
	gotExcludes [][]uuid.UUID
}

func newFakeScreenStore(videos ...*model.Video) *fakeScreenStore {
	s := &fakeScreenStore{
		videos:  make(map[uuid.UUID]*model.Video),
		pending: make(map[uuid.UUID]bool),
		status:  make(map[uuid.UUID]model.VideoStatus),
	}
	for _, v := range videos {
		v.ID = uuid.New()
		s.order = append(s.order, v.ID)
		s.videos[v.ID] = v
		s.pending[v.ID] = true
	}
	return s
}

func (s *fakeScreenStore) ListDiscoveredVideos(
	_ context.Context,
	limit int32,
	excludeIDs []uuid.UUID,
) ([]*model.Video, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.gotLimit = limit
	s.gotExcludes = append(s.gotExcludes, slices.Clone(excludeIDs))
	if s.listErr != nil {
		return nil, s.listErr
	}

	var out []*model.Video
	for _, id := range s.order {
		if int32(len(out)) >= limit {
			break
		}
		if s.pending[id] && !slices.Contains(excludeIDs, id) {
			out = append(out, s.videos[id])
		}
	}
	return out, nil
}

func (s *fakeScreenStore) SetVideoScreenStatus(
	_ context.Context,
	id uuid.UUID,
	status model.VideoStatus,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.setErrs[id]; err != nil {
		return false, err
	}
	if !s.pending[id] {
		return false, nil
	}
	s.pending[id] = false
	s.status[id] = status
	return true, nil
}

// fakeScreenAnalyzer is a data.ClaimAnalyzer that answers per text. It also
// records how many calls ran at the same time.
type fakeScreenAnalyzer struct {
	mu          sync.Mutex
	inScope     map[string]bool
	errs        map[string]error
	texts       []string
	topics      [][]data.AnalyzerTopic
	delay       time.Duration
	onAnalyze   func(text string)
	inflight    int
	maxInflight int
}

func (a *fakeScreenAnalyzer) Analyze(
	_ context.Context,
	message string,
	topics []data.AnalyzerTopic,
) (*data.ClaimAnalysis, error) {
	a.mu.Lock()
	a.texts = append(a.texts, message)
	a.topics = append(a.topics, topics)
	a.inflight++
	a.maxInflight = max(a.maxInflight, a.inflight)
	a.mu.Unlock()

	if a.onAnalyze != nil {
		a.onAnalyze(message)
	}
	time.Sleep(a.delay)

	a.mu.Lock()
	defer a.mu.Unlock()
	a.inflight--
	if err := a.errs[message]; err != nil {
		return nil, err
	}
	return &data.ClaimAnalysis{InScope: a.inScope[message]}, nil
}

func screenVideoFixture(caption string, hashtags ...string) *model.Video {
	return &model.Video{Caption: caption, Hashtags: hashtags}
}

func TestScreenText(t *testing.T) {
	tests := []struct {
		name  string
		video *model.Video
		want  string
	}{
		{"caption only", &model.Video{Caption: "obat herbal"}, "obat herbal"},
		{
			"caption with hashtags",
			&model.Video{Caption: "obat herbal", Hashtags: []string{"diabetes", "herbal"}},
			"obat herbal #diabetes #herbal",
		},
		{
			"caption is kept as it is",
			&model.Video{Caption: "  Obat   Herbal!  "},
			"  Obat   Herbal!  ",
		},
		{"hashtags only", &model.Video{Hashtags: []string{"santé"}}, " #santé"},
		{"caption and hashtags empty", &model.Video{}, ""},
		{"blank caption without hashtags", &model.Video{Caption: " \n\t "}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := screenText(tt.video); got != tt.want {
				t.Errorf("screenText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewScreenService_ConcurrencyIsAtLeastOne(t *testing.T) {
	for _, tt := range []struct{ in, want int }{{-5, 1}, {0, 1}, {1, 1}, {4, 4}} {
		t.Run(fmt.Sprintf("concurrency %d", tt.in), func(t *testing.T) {
			svc := NewScreenService(newFakeScreenStore(), &fakeScreenAnalyzer{}, tt.in)
			if svc.concurrency != tt.want {
				t.Errorf("concurrency = %d, want %d", svc.concurrency, tt.want)
			}
		})
	}
}

func TestScreenDiscoveredVideos_ScreensByScope(t *testing.T) {
	inScope := screenVideoFixture("obat herbal sembuhkan diabetes", "diabetes", "herbal")
	outOfScope := screenVideoFixture("resep nasi goreng", "kuliner")
	blank := screenVideoFixture("   ")
	store := newFakeScreenStore(inScope, outOfScope, blank)
	analyzer := &fakeScreenAnalyzer{inScope: map[string]bool{
		"obat herbal sembuhkan diabetes #diabetes #herbal": true,
	}}

	summary, err := NewScreenService(store, analyzer, 2).ScreenDiscoveredVideos(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := store.status[inScope.ID]; got != model.VideoStatusScreenedIn {
		t.Errorf("in-scope video status = %q, want screened_in", got)
	}
	if got := store.status[outOfScope.ID]; got != model.VideoStatusScreenedOut {
		t.Errorf("out-of-scope video status = %q, want screened_out", got)
	}
	if got := store.status[blank.ID]; got != model.VideoStatusScreenedOut {
		t.Errorf("blank video status = %q, want screened_out", got)
	}
	if len(analyzer.texts) != 2 {
		t.Errorf(
			"analyzer calls = %d, want 2 (blank text must not be analyzed)",
			len(analyzer.texts),
		)
	}
	if !slices.Contains(analyzer.texts, "obat herbal sembuhkan diabetes #diabetes #herbal") {
		t.Errorf("analyzer did not get the caption followed by the hashtags: %q", analyzer.texts)
	}
	for _, topics := range analyzer.topics {
		if topics != nil {
			t.Errorf("analyzer got topics %v, want nil", topics)
		}
	}
	if summary != (ScreenSummary{ScreenedIn: 1, ScreenedOut: 2}) {
		t.Errorf("summary = %+v", summary)
	}
}

func TestScreenDiscoveredVideos_NothingToScreen(t *testing.T) {
	analyzer := &fakeScreenAnalyzer{}

	summary, err := NewScreenService(newFakeScreenStore(), analyzer, 2).
		ScreenDiscoveredVideos(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary != (ScreenSummary{}) {
		t.Errorf("summary = %+v, want zero", summary)
	}
	if len(analyzer.texts) != 0 {
		t.Errorf("analyzer calls = %d, want 0", len(analyzer.texts))
	}
}

func TestScreenDiscoveredVideos_ProcessesEveryBatch(t *testing.T) {
	videos := make([]*model.Video, 0, screenBatchSize*2+20)
	for i := range screenBatchSize*2 + 20 {
		videos = append(videos, screenVideoFixture(fmt.Sprintf("health claim %d", i)))
	}
	store := newFakeScreenStore(videos...)

	summary, err := NewScreenService(store, &fakeScreenAnalyzer{}, 3).
		ScreenDiscoveredVideos(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.status) != len(videos) {
		t.Errorf("screened %d videos, want %d", len(store.status), len(videos))
	}
	if summary.ScreenedOut != len(videos) {
		t.Errorf("summary = %+v", summary)
	}
	if store.gotLimit != screenBatchSize {
		t.Errorf("batch size = %d, want %d", store.gotLimit, screenBatchSize)
	}
}

func TestScreenDiscoveredVideos_FailedVideoStaysDiscovered(t *testing.T) {
	good := screenVideoFixture("a valid claim")
	bad := screenVideoFixture("a failed claim")
	store := newFakeScreenStore(good, bad)
	analyzer := &fakeScreenAnalyzer{
		inScope: map[string]bool{"a valid claim": true},
		errs:    map[string]error{"a failed claim": errScreenBoom},
	}

	summary, err := NewScreenService(store, analyzer, 2).ScreenDiscoveredVideos(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := store.status[good.ID]; got != model.VideoStatusScreenedIn {
		t.Errorf("good video status = %q, want screened_in", got)
	}
	if !store.pending[bad.ID] {
		t.Error("failed video must stay discovered")
	}
	if summary.Failed != 1 || summary.ScreenedIn != 1 {
		t.Errorf("summary = %+v", summary)
	}

	// The failed video is excluded from later batches, or the run would
	// pick it up again and never finish.
	if len(store.gotExcludes) < 2 {
		t.Fatalf("expected at least two list calls, got %d", len(store.gotExcludes))
	}
	last := store.gotExcludes[len(store.gotExcludes)-1]
	if !slices.Contains(last, bad.ID) {
		t.Errorf("last list call excludes %v, want it to contain the failed video", last)
	}
}

func TestScreenDiscoveredVideos_StopsWhenAWholeBatchFails(t *testing.T) {
	a := screenVideoFixture("claim a")
	b := screenVideoFixture("claim b")
	store := newFakeScreenStore(a, b)
	analyzer := &fakeScreenAnalyzer{errs: map[string]error{
		"claim a": errScreenBoom,
		"claim b": errScreenBoom,
	}}

	summary, err := NewScreenService(store, analyzer, 2).ScreenDiscoveredVideos(t.Context())
	if !errors.Is(err, errScreenBoom) {
		t.Fatalf("error = %v, want it to wrap the analyzer error", err)
	}
	if summary.Failed != 2 {
		t.Errorf("summary = %+v, want 2 failed", summary)
	}
	if !store.pending[a.ID] || !store.pending[b.ID] {
		t.Error("videos must stay discovered after an outage")
	}
}

func TestScreenDiscoveredVideos_ListErrorStopsTheRun(t *testing.T) {
	store := newFakeScreenStore()
	store.listErr = errScreenBoom

	_, err := NewScreenService(store, &fakeScreenAnalyzer{}, 2).ScreenDiscoveredVideos(t.Context())
	if !errors.Is(err, errScreenBoom) {
		t.Fatalf("error = %v, want it to wrap the list error", err)
	}
}

func TestScreenDiscoveredVideos_StoreErrorCountsAsFailure(t *testing.T) {
	good := screenVideoFixture("a valid claim")
	bad := screenVideoFixture("claim that failed to be saved")
	store := newFakeScreenStore(good, bad)
	store.setErrs = map[uuid.UUID]error{bad.ID: errScreenBoom}

	summary, err := NewScreenService(store, &fakeScreenAnalyzer{}, 2).
		ScreenDiscoveredVideos(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, stored := store.status[good.ID]; !stored {
		t.Error("good video was not screened")
	}
	if !store.pending[bad.ID] {
		t.Error("video whose status could not be stored must stay discovered")
	}
	if summary.Failed != 1 || summary.ScreenedOut != 1 {
		t.Errorf("summary = %+v", summary)
	}
}

func TestScreenDiscoveredVideos_AlreadyScreenedIsNotAnError(t *testing.T) {
	video := screenVideoFixture("claim")
	store := newFakeScreenStore(video)
	analyzer := &fakeScreenAnalyzer{onAnalyze: func(string) {
		// Another run screens the same video while this one is analyzing it.
		store.mu.Lock()
		defer store.mu.Unlock()
		store.pending[video.ID] = false
	}}

	summary, err := NewScreenService(store, analyzer, 2).ScreenDiscoveredVideos(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Failed != 0 {
		t.Errorf("summary = %+v, want no failure", summary)
	}
	if _, stored := store.status[video.ID]; stored {
		t.Error("the late result must not overwrite the other run's status")
	}
}

func TestScreenDiscoveredVideos_CanceledContext(t *testing.T) {
	store := newFakeScreenStore(screenVideoFixture("claim"))
	analyzer := &fakeScreenAnalyzer{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := NewScreenService(store, analyzer, 2).ScreenDiscoveredVideos(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(analyzer.texts) != 0 {
		t.Errorf("analyzer calls = %d, want 0", len(analyzer.texts))
	}
}

func TestScreenDiscoveredVideos_RespectsConcurrencyLimit(t *testing.T) {
	for _, limit := range []int{1, 3} {
		t.Run(fmt.Sprintf("limit %d", limit), func(t *testing.T) {
			videos := make([]*model.Video, 0, 12)
			for i := range 12 {
				videos = append(videos, screenVideoFixture(fmt.Sprintf("claim %d", i)))
			}
			store := newFakeScreenStore(videos...)
			analyzer := &fakeScreenAnalyzer{delay: 10 * time.Millisecond}

			if _, err := NewScreenService(store, analyzer, limit).
				ScreenDiscoveredVideos(t.Context()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if analyzer.maxInflight > limit {
				t.Errorf(
					"max concurrent analyses = %d, want at most %d",
					analyzer.maxInflight,
					limit,
				)
			}
		})
	}
}
