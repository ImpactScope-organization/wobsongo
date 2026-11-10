package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

// crawlTestEnqueuer is a hand-rolled queue.JobEnqueuer.
type crawlTestEnqueuer struct {
	jobs []queue.BackgroundJob
	err  error
}

func (e *crawlTestEnqueuer) Enqueue(_ context.Context, payload queue.BackgroundJob) error {
	if e.err != nil {
		return e.err
	}
	e.jobs = append(e.jobs, payload)
	return nil
}

// screenJobs counts the screening jobs that were enqueued.
func (e *crawlTestEnqueuer) screenJobs() int {
	n := 0
	for _, job := range e.jobs {
		if _, ok := job.(queue.ScreenVideosJob); ok {
			n++
		}
	}
	return n
}

// crawlTestVideoRepo embeds data.VideoRepoer so only the methods a crawl
// uses need an implementation; calling any other one panics.
type crawlTestVideoRepo struct {
	data.VideoRepoer
	isNew     bool
	saveErrAt int // 1-based number of the save that fails, 0 for none
	calls     int
}

func (r *crawlTestVideoRepo) UpsertDiscoveredVideo(
	_ context.Context,
	v *model.Video,
) (bool, error) {
	r.calls++
	if r.saveErrAt == r.calls {
		return false, errors.New("db down")
	}
	v.ID = uuid.New()
	return r.isNew, nil
}

func (r *crawlTestVideoRepo) ListVideoTimesByInfluencerID(
	context.Context,
	uuid.UUID,
) ([]time.Time, error) {
	return nil, nil
}

// crawlTestScraper is a hand-rolled tikTokScraper.
type crawlTestScraper struct {
	items []dto.TikTokScraperItem
	err   error
}

func (s *crawlTestScraper) FetchProfileVideos(
	context.Context,
	string,
	int,
) ([]dto.TikTokScraperItem, error) {
	return s.items, s.err
}

func (s *crawlTestScraper) SearchByKeyword(
	context.Context,
	string,
	int,
) ([]dto.TikTokScraperItem, error) {
	return s.items, s.err
}

func crawlTestItem(id string) dto.TikTokScraperItem {
	return dto.TikTokScraperItem{
		ID:          id,
		Text:        "claim " + id,
		WebVideoURL: "https://www.tiktok.com/@creator/video/" + id,
		AuthorMeta:  dto.TikTokAuthorMeta{Name: "creator"},
	}
}

func newCrawlTestService(
	videos *crawlTestVideoRepo,
	scraper *crawlTestScraper,
	enqueuer queue.JobEnqueuer,
) *CrawlService {
	crawler := &mockrepo.CrawlerRepoerMock{
		GetSettingFunc: func(context.Context) (*model.AdminSetting, error) {
			return &model.AdminSetting{
				DiscoveryIntervalHours: 24,
				MaxResultsKeyword:      5,
				MaxResultsInfluencer:   5,
			}, nil
		},
		ListActiveKeywordsFunc: func(context.Context) ([]*model.DiscoveryKeyword, error) {
			return []*model.DiscoveryKeyword{{ID: uuid.New(), Keyword: "contraception"}}, nil
		},
		FailStaleRunsFunc: func(context.Context, model.CrawlKind, time.Duration) error {
			return nil
		},
		CreateRunFunc: func(_ context.Context, run *model.CrawlRun) error {
			run.ID = uuid.New()
			return nil
		},
		MarkKeywordRunFunc: func(context.Context, uuid.UUID) error { return nil },
		FinishRunFunc:      func(context.Context, *model.CrawlRun) error { return nil },
	}
	influencers := &mockrepo.InfluencerRepoerMock{
		GetByIDFunc: func(_ context.Context, id uuid.UUID) (*model.Influencer, error) {
			return &model.Influencer{ID: id, Username: "creator"}, nil
		},
		UpsertInfluencerFunc: func(_ context.Context, influencer *model.Influencer) error {
			influencer.ID = uuid.New()
			return nil
		},
		UpdateAggregatesFunc: func(
			context.Context, uuid.UUID, *time.Time, *time.Time, *float64, int64,
		) error {
			return nil
		},
		SetNextCheckAtFunc: func(context.Context, uuid.UUID, time.Time) error { return nil },
	}
	return NewCrawlService(crawler, influencers, videos, scraper, enqueuer)
}

func TestCrawlService_enqueueScreening(t *testing.T) {
	t.Run("NoNewVideos_EnqueuesNothing", func(t *testing.T) {
		enqueuer := &crawlTestEnqueuer{}
		svc := &CrawlService{screenEnqueuer: enqueuer}

		svc.enqueueScreening(t.Context(), 0)

		if len(enqueuer.jobs) != 0 {
			t.Errorf("enqueued %d jobs, want 0", len(enqueuer.jobs))
		}
	})

	t.Run("NewVideos_EnqueuesOneScreeningJob", func(t *testing.T) {
		enqueuer := &crawlTestEnqueuer{}
		svc := &CrawlService{screenEnqueuer: enqueuer}

		svc.enqueueScreening(t.Context(), 7)

		if len(enqueuer.jobs) != 1 || enqueuer.screenJobs() != 1 {
			t.Errorf("enqueued %v, want exactly one ScreenVideosJob", enqueuer.jobs)
		}
	})

	t.Run("EnqueueFailure_IsOnlyLogged", func(t *testing.T) {
		enqueuer := &crawlTestEnqueuer{err: errors.New("queue unavailable")}
		svc := &CrawlService{screenEnqueuer: enqueuer}

		svc.enqueueScreening(t.Context(), 3) // must not panic or fail the caller

		if len(enqueuer.jobs) != 0 {
			t.Errorf("enqueued %d jobs, want 0", len(enqueuer.jobs))
		}
	})

	t.Run("NoEnqueuer_IsSkipped", func(t *testing.T) {
		svc := &CrawlService{}

		svc.enqueueScreening(t.Context(), 3) // must not panic
	})
}

func TestCrawlService_RunDiscovery_EnqueuesScreening(t *testing.T) {
	tests := []struct {
		name     string
		videos   *crawlTestVideoRepo
		scraper  *crawlTestScraper
		wantJobs int
		wantErr  bool
	}{
		{
			"new video",
			&crawlTestVideoRepo{isNew: true},
			&crawlTestScraper{items: []dto.TikTokScraperItem{crawlTestItem("1")}},
			1,
			false,
		},
		{
			"video already known",
			&crawlTestVideoRepo{isNew: false},
			&crawlTestScraper{items: []dto.TikTokScraperItem{crawlTestItem("1")}},
			0,
			false,
		},
		{
			"nothing found",
			&crawlTestVideoRepo{isNew: true},
			&crawlTestScraper{},
			0,
			false,
		},
		{
			"scraper fails",
			&crawlTestVideoRepo{isNew: true},
			&crawlTestScraper{err: errors.New("apify down")},
			0,
			true,
		},
		{
			// The run is reported as failed, but the first video was
			// stored, so it still has to be screened.
			"run fails after storing a new video",
			&crawlTestVideoRepo{isNew: true, saveErrAt: 2},
			&crawlTestScraper{
				items: []dto.TikTokScraperItem{crawlTestItem("1"), crawlTestItem("2")},
			},
			1,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enqueuer := &crawlTestEnqueuer{}
			svc := newCrawlTestService(tt.videos, tt.scraper, enqueuer)

			err := svc.RunDiscovery(t.Context(), model.CrawlTriggerManual, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := enqueuer.screenJobs(); got != tt.wantJobs {
				t.Errorf("screening jobs = %d, want %d", got, tt.wantJobs)
			}
		})
	}
}

func TestCrawlService_RunDiscovery_EnqueueFailureDoesNotFailTheRun(t *testing.T) {
	enqueuer := &crawlTestEnqueuer{err: errors.New("queue unavailable")}
	svc := newCrawlTestService(
		&crawlTestVideoRepo{isNew: true},
		&crawlTestScraper{items: []dto.TikTokScraperItem{crawlTestItem("1")}},
		enqueuer,
	)

	if err := svc.RunDiscovery(t.Context(), model.CrawlTriggerManual, nil); err != nil {
		t.Fatalf("a failed enqueue must not fail the run, got %v", err)
	}
}

func TestCrawlService_RunMonitor_EnqueuesScreening(t *testing.T) {
	tests := []struct {
		name     string
		videos   *crawlTestVideoRepo
		wantJobs int
	}{
		{"new video", &crawlTestVideoRepo{isNew: true}, 1},
		{"video already known", &crawlTestVideoRepo{isNew: false}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enqueuer := &crawlTestEnqueuer{}
			scraper := &crawlTestScraper{items: []dto.TikTokScraperItem{crawlTestItem("1")}}
			svc := newCrawlTestService(tt.videos, scraper, enqueuer)
			creatorID := uuid.New()

			err := svc.RunMonitor(t.Context(), model.CrawlTriggerManual, nil, &creatorID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := enqueuer.screenJobs(); got != tt.wantJobs {
				t.Errorf("screening jobs = %d, want %d", got, tt.wantJobs)
			}
		})
	}
}
