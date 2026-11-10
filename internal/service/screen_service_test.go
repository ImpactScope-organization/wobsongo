package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

var errScreenBoom = errors.New("llm unavailable")

// fakeScreenStore is a data.VideoRepoer that keeps the videos in memory and records the calls made to it.
type fakeScreenStore struct {
	data.VideoRepoer
	order       []uuid.UUID
	videos      map[uuid.UUID]*model.Video
	pending     map[uuid.UUID]bool
	status      map[uuid.UUID]model.VideoStatus
	enqueued    []uuid.UUID
	listErr     error
	getErr      error
	setErrs     map[uuid.UUID]error
	enqueueErrs map[uuid.UUID]error
	duplicates  map[uuid.UUID]bool
	gotLimit    int32
	gotExcludes [][]uuid.UUID
	failed      map[uuid.UUID]string
	failErrs    map[uuid.UUID]error
}

func (s *fakeScreenStore) SetVideoScreenFailed(
	_ context.Context,
	id uuid.UUID,
	reason string,
) (bool, error) {
	if err := s.failErrs[id]; err != nil {
		return false, err
	}
	if !s.pending[id] {
		return false, nil
	}
	s.pending[id] = false
	s.status[id] = model.VideoStatusFailed
	if s.failed == nil {
		s.failed = make(map[uuid.UUID]string)
	}
	s.failed[id] = reason
	return true, nil
}

func newFakeScreenStore(videos ...*model.Video) *fakeScreenStore {
	s := &fakeScreenStore{
		videos:   make(map[uuid.UUID]*model.Video),
		pending:  make(map[uuid.UUID]bool),
		status:   make(map[uuid.UUID]model.VideoStatus),
		failed:   make(map[uuid.UUID]string),
		failErrs: make(map[uuid.UUID]error),
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
	after time.Time,
	afterID uuid.UUID,
) ([]*model.Video, error) {
	s.gotLimit = limit
	start := 0
	if afterID != uuid.Nil {
		if i := slices.Index(s.order, afterID); i >= 0 {
			start = i + 1
		}
	}
	s.gotExcludes = append(s.gotExcludes, slices.Clone(s.order[:start]))
	if s.listErr != nil {
		return nil, s.listErr
	}

	var out []*model.Video
	for _, id := range s.order[start:] {
		if int32(len(out)) >= limit {
			break
		}
		if s.pending[id] {
			out = append(out, s.videos[id])
		}
	}
	return out, nil
}

func (s *fakeScreenStore) GetVideoForScreening(
	_ context.Context,
	id uuid.UUID,
) (*model.Video, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if !s.pending[id] {
		return nil, data.ErrNotFound
	}
	return s.videos[id], nil
}

func (s *fakeScreenStore) SetVideoScreenStatus(
	_ context.Context,
	id uuid.UUID,
	status model.VideoStatus,
) (bool, error) {
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

func (s *fakeScreenStore) EnqueueScreenVideoJob(
	_ context.Context,
	videoID uuid.UUID,
) (bool, error) {
	if err := s.enqueueErrs[videoID]; err != nil {
		return false, err
	}
	if s.duplicates[videoID] {
		return false, nil
	}
	s.enqueued = append(s.enqueued, videoID)
	return true, nil
}

// fakeScreenAnalyzer is a data.ClaimAnalyzer that answers per text.
type fakeScreenAnalyzer struct {
	inScope    map[string]bool
	errs       map[string]error
	noAnalysis bool
	onAnalyze  func(text string)
	texts      []string
	topics     [][]data.AnalyzerTopic
}

func (a *fakeScreenAnalyzer) Analyze(
	_ context.Context,
	message string,
	topics []data.AnalyzerTopic,
) (*data.ClaimAnalysis, error) {
	a.texts = append(a.texts, message)
	a.topics = append(a.topics, topics)

	if a.onAnalyze != nil {
		a.onAnalyze(message)
	}
	if err := a.errs[message]; err != nil {
		return nil, err
	}
	if a.noAnalysis {
		return nil, nil
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
		{"caption only", &model.Video{Caption: "herbal remedy"}, "herbal remedy"},
		{
			"caption with hashtags",
			&model.Video{Caption: "herbal remedy", Hashtags: []string{"diabetes", "herbal"}},
			"herbal remedy #diabetes #herbal",
		},
		{
			"caption is kept as it is",
			&model.Video{Caption: "  Herbal   Remedy!  "},
			"  Herbal   Remedy!  ",
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

func TestScreenVideo_ScreensByScope(t *testing.T) {
	tests := []struct {
		name      string
		video     *model.Video
		inScope   map[string]bool
		want      model.VideoStatus
		wantCalls int
	}{
		{
			name:      "in scope video is screened in",
			video:     screenVideoFixture("herbal remedy cures diabetes", "diabetes", "herbal"),
			inScope:   map[string]bool{"herbal remedy cures diabetes #diabetes #herbal": true},
			want:      model.VideoStatusScreenedIn,
			wantCalls: 1,
		},
		{
			name:      "out of scope video is screened out",
			video:     screenVideoFixture("pasta recipe", "cooking"),
			want:      model.VideoStatusScreenedOut,
			wantCalls: 1,
		},
		{
			name:  "blank video is screened out without calling the analyzer",
			video: screenVideoFixture("   "),
			want:  model.VideoStatusScreenedOut,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeScreenStore(tt.video)
			analyzer := &fakeScreenAnalyzer{inScope: tt.inScope}

			svc := NewScreenService(store, analyzer)
			if err := svc.ScreenVideo(context.Background(), tt.video.ID); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := store.status[tt.video.ID]; got != tt.want {
				t.Errorf("status = %q, want %q", got, tt.want)
			}
			if len(analyzer.texts) != tt.wantCalls {
				t.Errorf("analyzer calls = %d, want %d", len(analyzer.texts), tt.wantCalls)
			}
			for _, topics := range analyzer.topics {
				if topics != nil {
					t.Errorf("analyzer got topics %v, want nil", topics)
				}
			}
		})
	}
}

func TestScreenVideo_AnalyzerErrorKeepsVideoDiscovered(t *testing.T) {
	video := screenVideoFixture("a failed claim")
	store := newFakeScreenStore(video)
	analyzer := &fakeScreenAnalyzer{errs: map[string]error{"a failed claim": errScreenBoom}}

	err := NewScreenService(store, analyzer).ScreenVideo(context.Background(), video.ID)
	if !errors.Is(err, errScreenBoom) {
		t.Fatalf("error = %v, want it to wrap the analyzer error", err)
	}
	if !store.pending[video.ID] {
		t.Error("video must stay discovered after an analyzer error")
	}
}

func TestScreenVideo_NilAnalysisIsAnError(t *testing.T) {
	video := screenVideoFixture("a claim")
	store := newFakeScreenStore(video)
	analyzer := &fakeScreenAnalyzer{noAnalysis: true}

	err := NewScreenService(store, analyzer).ScreenVideo(context.Background(), video.ID)
	if !errors.Is(err, errNoAnalysis) {
		t.Fatalf("error = %v, want it to wrap errNoAnalysis", err)
	}
	if !store.pending[video.ID] {
		t.Error("video without an analysis must stay discovered")
	}
}

func TestScreenVideo_StoreErrorIsReturned(t *testing.T) {
	video := screenVideoFixture("a claim")
	store := newFakeScreenStore(video)
	store.setErrs = map[uuid.UUID]error{video.ID: errScreenBoom}

	err := NewScreenService(
		store,
		&fakeScreenAnalyzer{},
	).ScreenVideo(context.Background(), video.ID)
	if !errors.Is(err, errScreenBoom) {
		t.Fatalf("error = %v, want it to wrap the store error", err)
	}
	if !store.pending[video.ID] {
		t.Error("video whose status could not be stored must stay discovered")
	}
}

func TestScreenVideo_GetErrorIsReturned(t *testing.T) {
	store := newFakeScreenStore()
	store.getErr = errScreenBoom
	analyzer := &fakeScreenAnalyzer{}

	err := NewScreenService(store, analyzer).ScreenVideo(context.Background(), uuid.New())
	if !errors.Is(err, errScreenBoom) {
		t.Fatalf("error = %v, want it to wrap the get error", err)
	}
	if len(analyzer.texts) != 0 {
		t.Errorf("analyzer calls = %d, want 0", len(analyzer.texts))
	}
}

func TestScreenVideo_UnknownVideoIsNotAnError(t *testing.T) {
	analyzer := &fakeScreenAnalyzer{}

	err := NewScreenService(
		newFakeScreenStore(),
		analyzer,
	).ScreenVideo(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(analyzer.texts) != 0 {
		t.Errorf("analyzer calls = %d, want 0", len(analyzer.texts))
	}
}

func TestScreenVideo_AlreadyScreenedIsNotAnError(t *testing.T) {
	video := screenVideoFixture("claim")
	store := newFakeScreenStore(video)
	analyzer := &fakeScreenAnalyzer{onAnalyze: func(string) {
		// Another run screens the same video while this one is analyzing it.
		store.pending[video.ID] = false
	}}

	err := NewScreenService(store, analyzer).ScreenVideo(context.Background(), video.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, stored := store.status[video.ID]; stored {
		t.Error("the late result must not overwrite the other run's status")
	}
}

func TestEnqueueDiscoveredVideos_NothingToEnqueue(t *testing.T) {
	store := newFakeScreenStore()

	count, err := NewScreenService(store, &fakeScreenAnalyzer{}).
		EnqueueDiscoveredVideos(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 || len(store.enqueued) != 0 {
		t.Errorf("count = %d, enqueued = %d, want none", count, len(store.enqueued))
	}
}

func TestEnqueueDiscoveredVideos_EnqueuesEveryBatch(t *testing.T) {
	total := screenBatchSize*2 + 20
	videos := make([]*model.Video, 0, total)
	for i := range total {
		videos = append(videos, screenVideoFixture(fmt.Sprintf("health claim %d", i)))
	}
	store := newFakeScreenStore(videos...)
	analyzer := &fakeScreenAnalyzer{}

	count, err := NewScreenService(store, analyzer).EnqueueDiscoveredVideos(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != total || len(store.enqueued) != total {
		t.Errorf("count = %d, enqueued = %d, want %d", count, len(store.enqueued), total)
	}
	unique := make(map[uuid.UUID]bool, total)
	for _, id := range store.enqueued {
		unique[id] = true
	}
	if len(unique) != total {
		t.Errorf("enqueued %d distinct videos, want %d", len(unique), total)
	}
	if store.gotLimit != screenBatchSize {
		t.Errorf("batch size = %d, want %d", store.gotLimit, screenBatchSize)
	}
	// Videos stay discovered until their job runs, so each list call has to
	// skip the ones already enqueued, or the run would never finish.
	last := store.gotExcludes[len(store.gotExcludes)-1]
	if len(last) != total {
		t.Errorf("last list call excludes %d videos, want %d", len(last), total)
	}
	if len(analyzer.texts) != 0 {
		t.Errorf("analyzer calls = %d, want 0 (screening happens in the jobs)", len(analyzer.texts))
	}
}

func TestEnqueueDiscoveredVideos_ListErrorStopsTheRun(t *testing.T) {
	store := newFakeScreenStore()
	store.listErr = errScreenBoom

	_, err := NewScreenService(store, &fakeScreenAnalyzer{}).EnqueueDiscoveredVideos(t.Context())
	if !errors.Is(err, errScreenBoom) {
		t.Fatalf("error = %v, want it to wrap the list error", err)
	}
}

func TestEnqueueDiscoveredVideos_EnqueueErrorStopsTheRun(t *testing.T) {
	a := screenVideoFixture("claim a")
	b := screenVideoFixture("claim b")
	c := screenVideoFixture("claim c")
	store := newFakeScreenStore(a, b, c)
	store.enqueueErrs = map[uuid.UUID]error{b.ID: errScreenBoom}

	count, err := NewScreenService(store, &fakeScreenAnalyzer{}).
		EnqueueDiscoveredVideos(context.Background())
	if !errors.Is(err, errScreenBoom) {
		t.Fatalf("error = %v, want it to wrap the enqueue error", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (only the video before the failure)", count)
	}
	if want := []uuid.UUID{a.ID}; !slices.Equal(store.enqueued, want) {
		t.Errorf("enqueued = %v, want %v", store.enqueued, want)
	}
}

func TestEnqueueDiscoveredVideos_CanceledContext(t *testing.T) {
	store := newFakeScreenStore(screenVideoFixture("claim"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewScreenService(store, &fakeScreenAnalyzer{}).EnqueueDiscoveredVideos(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(store.enqueued) != 0 {
		t.Errorf("enqueued = %d, want 0", len(store.enqueued))
	}
}

func TestEnqueueDiscoveredVideos_DuplicateJobIsNotCounted(t *testing.T) {
	a := screenVideoFixture("claim a")
	b := screenVideoFixture("claim b")
	store := newFakeScreenStore(a, b)
	store.duplicates = map[uuid.UUID]bool{a.ID: true}

	count, err := NewScreenService(store, &fakeScreenAnalyzer{}).
		EnqueueDiscoveredVideos(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (the duplicate is not counted)", count)
	}
	if want := []uuid.UUID{b.ID}; !slices.Equal(store.enqueued, want) {
		t.Errorf("enqueued = %v, want %v", store.enqueued, want)
	}
}

func TestMarkScreenFailed(t *testing.T) {
	t.Run("MarksDiscoveredVideoAsFailed", func(t *testing.T) {
		video := screenVideoFixture("claim")
		store := newFakeScreenStore(video)

		err := NewScreenService(store, &fakeScreenAnalyzer{}).
			MarkScreenFailed(context.Background(), video.ID, errScreenBoom)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := store.status[video.ID]; got != model.VideoStatusFailed {
			t.Errorf("status = %q, want failed", got)
		}
		if got := store.failed[video.ID]; got != errScreenBoom.Error() {
			t.Errorf("reason = %q, want %q", got, errScreenBoom.Error())
		}
	})

	t.Run("CutsLongReason", func(t *testing.T) {
		video := screenVideoFixture("claim")
		store := newFakeScreenStore(video)
		long := errors.New(strings.Repeat("é", maxFailureReasonLen*2))

		err := NewScreenService(store, &fakeScreenAnalyzer{}).
			MarkScreenFailed(context.Background(), video.ID, long)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := len([]rune(store.failed[video.ID])); got != maxFailureReasonLen {
			t.Errorf("reason length = %d, want %d", got, maxFailureReasonLen)
		}
	})

	t.Run("AlreadyScreenedIsNotAnError", func(t *testing.T) {
		video := screenVideoFixture("claim")
		store := newFakeScreenStore(video)
		store.pending[video.ID] = false

		err := NewScreenService(store, &fakeScreenAnalyzer{}).
			MarkScreenFailed(context.Background(), video.ID, errScreenBoom)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("StoreErrorIsReturned", func(t *testing.T) {
		video := screenVideoFixture("claim")
		store := newFakeScreenStore(video)
		store.failErrs = map[uuid.UUID]error{video.ID: errScreenBoom}

		err := NewScreenService(store, &fakeScreenAnalyzer{}).
			MarkScreenFailed(context.Background(), video.ID, errors.New("analyze"))
		if !errors.Is(err, errScreenBoom) {
			t.Fatalf("error = %v, want it to wrap the store error", err)
		}
	})
}
