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
)

// crawlTestVideoRepo embeds data.VideoRepoer so only the methods a crawl
// uses need an implementation; calling any other one panics.
type crawlTestVideoRepo struct {
	data.VideoRepoer
	isNew            bool
	saveErrAt        int // 1-based number of the save that fails, 0 for none
	calls            int
	screenRuns       int   // how many times EnqueueScreenVideosJob was called
	enqueueScreenErr error // returned by EnqueueScreenVideosJob
}

func (r *crawlTestVideoRepo) EnqueueScreenVideosJob(context.Context) error {
	r.screenRuns++
	return r.enqueueScreenErr
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
	return NewCrawlService(crawler, influencers, videos, scraper)
}

func TestCrawlService_enqueueScreening(t *testing.T) {
	t.Run("NoNewVideos_EnqueuesNothing", func(t *testing.T) {
		videos := &crawlTestVideoRepo{}
		svc := &CrawlService{videoRepo: videos}

		svc.enqueueScreening(t.Context(), 0)

		if videos.screenRuns != 0 {
			t.Errorf("screening runs = %d, want 0", videos.screenRuns)
		}
	})

	t.Run("NewVideos_EnqueuesOneScreeningJob", func(t *testing.T) {
		videos := &crawlTestVideoRepo{}
		svc := &CrawlService{videoRepo: videos}

		svc.enqueueScreening(t.Context(), 7)

		if videos.screenRuns != 1 {
			t.Errorf("screening runs = %d, want 1", videos.screenRuns)
		}
	})

	t.Run("EnqueueFailure_IsOnlyLogged", func(t *testing.T) {
		videos := &crawlTestVideoRepo{enqueueScreenErr: errors.New("queue unavailable")}
		svc := &CrawlService{videoRepo: videos}

		svc.enqueueScreening(t.Context(), 3) // must not panic or fail the caller

		if videos.screenRuns != 1 {
			t.Errorf("screening runs = %d, want 1", videos.screenRuns)
		}
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
			svc := newCrawlTestService(tt.videos, tt.scraper)

			err := svc.RunDiscovery(t.Context(), model.CrawlTriggerManual, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := tt.videos.screenRuns; got != tt.wantJobs {
				t.Errorf("screening runs = %d, want %d", got, tt.wantJobs)
			}
		})
	}
}

func TestCrawlService_RunDiscovery_EnqueueFailureDoesNotFailTheRun(t *testing.T) {
	videos := &crawlTestVideoRepo{
		isNew:            true,
		enqueueScreenErr: errors.New("queue unavailable"),
	}
	svc := newCrawlTestService(
		videos,
		&crawlTestScraper{items: []dto.TikTokScraperItem{crawlTestItem("1")}},
	)

	if err := svc.RunDiscovery(t.Context(), model.CrawlTriggerManual, nil); err != nil {
		t.Fatalf("a failed enqueue must not fail the run, got %v", err)
	}
	if videos.screenRuns != 1 {
		t.Errorf("screening runs = %d, want 1", videos.screenRuns)
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
			scraper := &crawlTestScraper{items: []dto.TikTokScraperItem{crawlTestItem("1")}}
			svc := newCrawlTestService(tt.videos, scraper)
			creatorID := uuid.New()

			err := svc.RunMonitor(t.Context(), model.CrawlTriggerManual, nil, &creatorID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := tt.videos.screenRuns; got != tt.wantJobs {
				t.Errorf("screening runs = %d, want %d", got, tt.wantJobs)
			}
		})
	}
}
