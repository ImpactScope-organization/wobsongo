package model

import (
	"time"

	"github.com/google/uuid"
)

// DiscoveryKeyword is a search keyword used to discover new videos.
type DiscoveryKeyword struct {
	ID           uuid.UUID  `json:"id"                    format:"uuid"`
	Keyword      string     `json:"keyword"`
	Language     Language   `json:"language"`
	LanguageCode string     `json:"language_code"`
	IsActive     bool       `json:"is_active"`
	LastRunAt    *time.Time `json:"last_run_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// AdminSetting is the admin-managed configuration for discovery and
// monitoring. There is always exactly one.
type AdminSetting struct {
	DiscoveryIntervalHours int `json:"discovery_interval_hours"`

	// MaxResultsKeyword is how many videos to fetch per discovery keyword.
	MaxResultsKeyword int `json:"max_results_keyword"`

	// MaxResultsInfluencer is how many recent videos to fetch per creator check.
	MaxResultsInfluencer int `json:"max_results_influencer"`

	// MaxCostPerRun is the maximum cost (USD) allowed for a single discovery or
	// monitoring run. If the Apify actor reports a higher cost, the run is
	// considered failed.
	MaxCostPerRun float64    `json:"max_cost_per_run"`
	UpdatedBy     *uuid.UUID `json:"updated_by,omitempty" format:"uuid"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// CrawlKind is the kind of a crawl run.
type CrawlKind string

const (
	CrawlKindDiscover CrawlKind = "discover"
	CrawlKindMonitor  CrawlKind = "monitor"
)

// CrawlTrigger tells who started a crawl run.
type CrawlTrigger string

const (
	CrawlTriggerScheduled CrawlTrigger = "scheduled"
	CrawlTriggerManual    CrawlTrigger = "manual"
)

// CrawlStatus is the state of a crawl run.
type CrawlStatus string

const (
	CrawlStatusRunning   CrawlStatus = "running"
	CrawlStatusSucceeded CrawlStatus = "succeeded"
	CrawlStatusFailed    CrawlStatus = "failed"
)

// CrawlRun is the history of one discovery or monitoring run.
type CrawlRun struct {
	ID          uuid.UUID    `json:"id"                     format:"uuid"`
	Kind        CrawlKind    `json:"kind"`
	Trigger     CrawlTrigger `json:"trigger"`
	TriggeredBy *uuid.UUID   `json:"triggered_by,omitempty" format:"uuid"`
	Status      CrawlStatus  `json:"status"`
	StartedAt   time.Time    `json:"started_at"`
	FinishedAt  *time.Time   `json:"finished_at,omitempty"`
	VideoFound  int          `json:"video_found"`
	VideoNew    int          `json:"video_new"`
	Cost        *float64     `json:"cost,omitempty"`

	// Params is a snapshot (JSON) of the keywords and max results used, so the
	// history stays accurate when an admin changes them later.
	Params []byte `json:"params"`

	Error *string `json:"error,omitempty"`
}
