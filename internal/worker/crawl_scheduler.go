package worker

import (
	"sync"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// CrawlScheduler owns the periodic schedule of the discovery job, so the
// interval can change at runtime without restarting the server.
type CrawlScheduler struct {
	clientFn  func() *river.Client[pgx.Tx]
	mu        sync.Mutex
	handle    rivertype.PeriodicJobHandle
	hasHandle bool
}

// NewCrawlScheduler creates a CrawlScheduler.
func NewCrawlScheduler(clientFn func() *river.Client[pgx.Tx]) *CrawlScheduler {
	return &CrawlScheduler{clientFn: clientFn}
}

// Reschedule replaces the discovery schedule with one that runs every
// intervalHours. runOnStart makes the first run happen right away; it is
// true at server start and false when an admin changes the interval.
func (s *CrawlScheduler) Reschedule(intervalHours int, runOnStart bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	periodic := s.clientFn().PeriodicJobs()
	if s.hasHandle {
		periodic.Remove(s.handle)
	}

	s.handle = periodic.Add(river.NewPeriodicJob(
		river.PeriodicInterval(time.Duration(intervalHours)*time.Hour),
		func() (river.JobArgs, *river.InsertOpts) {
			return queue.DiscoverVideosJob{Trigger: queue.CrawlTriggerScheduled}, nil
		},
		&river.PeriodicJobOpts{RunOnStart: runOnStart},
	))
	s.hasHandle = true
}
