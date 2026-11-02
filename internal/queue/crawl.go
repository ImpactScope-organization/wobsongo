package queue

import (
	"time"

	"github.com/riverqueue/river"
)

const (
	// MonitorCheckInterval is how often River looks for creators that are
	// due.
	MonitorCheckInterval = 30 * time.Minute

	// CrawlerMaxWorkers allows one discovery and one monitor run at a time.
	CrawlerMaxWorkers = 2

	// CrawlTriggerScheduled marks a run started by the periodic tick.
	CrawlTriggerScheduled = "scheduled"

	// CrawlTriggerManual marks a run started by an admin.
	CrawlTriggerManual = "manual"
)

// DiscoverVideosJob runs one video discovery over every active keyword.
type DiscoverVideosJob struct {
	// Trigger is "scheduled" or "manual". A scheduled run does nothing
	// unless the discovery interval has passed.
	Trigger string `json:"trigger"`

	// TriggeredBy is the admin (users.id) who started a manual run.
	TriggeredBy string `json:"triggered_by,omitempty"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (DiscoverVideosJob) Kind() string { return string(JobTypeDiscoverVideos) }

// InsertOpts implements river.JobArgsWithInsertOpts. There is no retry:
// retrying would spend Apify credits again, and the failure is already
// recorded in crawl_runs.
func (DiscoverVideosJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueCrawler, MaxAttempts: 1}
}

// MonitorCreatorJob checks tracked creators for new videos.
type MonitorCreatorJob struct {
	// Trigger is "scheduled" or "manual".
	Trigger string `json:"trigger"`

	// TriggeredBy is the admin who started a manual run.
	TriggeredBy string `json:"triggered_by,omitempty"`

	// InfluencerID limits the run to one creator (admin "Check Now"). Empty
	// means every creator that is due.
	InfluencerID string `json:"influencer_id,omitempty"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (MonitorCreatorJob) Kind() string { return string(JobTypeMonitorCreator) }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (MonitorCreatorJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueCrawler, MaxAttempts: 1}
}
