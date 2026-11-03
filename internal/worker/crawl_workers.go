package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

// crawlJobTimeout sets the maximum duration for an entire crawl job.
const crawlJobTimeout = 30 * time.Minute

// DiscoverVideosWorker runs the keyword-based video discovery.
type DiscoverVideosWorker struct {
	river.WorkerDefaults[queue.DiscoverVideosJob]
	crawlService *service.CrawlService
}

// NewDiscoverVideosWorker creates a new DiscoverVideosWorker.
func NewDiscoverVideosWorker(crawlService *service.CrawlService) *DiscoverVideosWorker {
	return &DiscoverVideosWorker{crawlService: crawlService}
}

func (w *DiscoverVideosWorker) Timeout(_ *river.Job[queue.DiscoverVideosJob]) time.Duration {
	return crawlJobTimeout
}

// Work runs one discovery.
func (w *DiscoverVideosWorker) Work(
	ctx context.Context,
	job *river.Job[queue.DiscoverVideosJob],
) error {
	trigger, triggeredBy, err := parseCrawlArgs(job.Args.Trigger, job.Args.TriggeredBy)
	if err != nil {
		return river.JobCancel(err)
	}
	if err := w.crawlService.RunDiscovery(ctx, trigger, triggeredBy); err != nil {
		return fmt.Errorf("failed to discover videos: %w", err)
	}
	return nil
}

// MonitorCreatorWorker checks tracked creators for new videos.
type MonitorCreatorWorker struct {
	river.WorkerDefaults[queue.MonitorCreatorJob]
	crawlService *service.CrawlService
}

// NewMonitorCreatorWorker creates a new MonitorCreatorWorker.
func NewMonitorCreatorWorker(crawlService *service.CrawlService) *MonitorCreatorWorker {
	return &MonitorCreatorWorker{crawlService: crawlService}
}

func (w *MonitorCreatorWorker) Timeout(_ *river.Job[queue.MonitorCreatorJob]) time.Duration {
	return crawlJobTimeout
}

// Work runs one monitor pass.
func (w *MonitorCreatorWorker) Work(
	ctx context.Context,
	job *river.Job[queue.MonitorCreatorJob],
) error {
	trigger, triggeredBy, err := parseCrawlArgs(job.Args.Trigger, job.Args.TriggeredBy)
	if err != nil {
		return river.JobCancel(err)
	}

	var influencerID *uuid.UUID
	if job.Args.InfluencerID != "" {
		id, err := uuid.Parse(job.Args.InfluencerID)
		if err != nil {
			return river.JobCancel(
				fmt.Errorf("invalid influencer id %q: %w", job.Args.InfluencerID, err),
			)
		}
		influencerID = &id
	}

	if err := w.crawlService.RunMonitor(ctx, trigger, triggeredBy, influencerID); err != nil {
		return fmt.Errorf("failed to monitor creators: %w", err)
	}
	return nil
}

// parseCrawlArgs validates the arguments shared by both crawl jobs.
func parseCrawlArgs(trigger, triggeredBy string) (model.CrawlTrigger, *uuid.UUID, error) {
	t := model.CrawlTrigger(trigger)
	if t != model.CrawlTriggerScheduled && t != model.CrawlTriggerManual {
		return "", nil, fmt.Errorf("invalid crawl trigger %q", trigger)
	}

	if triggeredBy == "" {
		return t, nil, nil
	}
	id, err := uuid.Parse(triggeredBy)
	if err != nil {
		return "", nil, fmt.Errorf("invalid triggered_by %q: %w", triggeredBy, err)
	}
	return t, &id, nil
}
