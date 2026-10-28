package queue

import (
	"time"

	"github.com/riverqueue/river"
)

const (
	// DefaultNarrativeActiveWindowHours is how long (in hours) a narrative
	// stays active without new claims, unless NARRATIVE_ACTIVE_WINDOW_HOURS
	// says otherwise.
	DefaultNarrativeActiveWindowHours = 168

	// DefaultNarrativeMatchThreshold is the default minimum cosine similarity
	// for a claim to join an existing narrative. It depends on the embedding model.
	DefaultNarrativeMatchThreshold = 0.7

	// NarrativeDeactivateInterval is how often the deactivation job runs.
	NarrativeDeactivateInterval = time.Hour

	// DefaultNarrativeClusteringMaxWorkers is how many cluster-claim jobs may
	// run at once.
	DefaultNarrativeClusteringMaxWorkers = 2
)

// ClusterClaimJob groups one stored claim into a narrative of its topic.
type ClusterClaimJob struct {
	// ClaimID is the claims.id of the claim to group.
	ClaimID string `json:"claim_id"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (ClusterClaimJob) Kind() string { return string(JobTypeClusterClaim) }

// InsertOpts implements river.JobArgsWithInsertOpts. A claim is only grouped
// once even if the job is enqueued twice.
func (ClusterClaimJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:      QueueNarrativeClustering,
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	}
}

// DeactivateNarrativesJob sets is_active to false on narratives that had no
// claim inside the activity window. It is enqueued periodically by River.
type DeactivateNarrativesJob struct{}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (DeactivateNarrativesJob) Kind() string { return string(JobTypeDeactivateNarratives) }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (DeactivateNarrativesJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueMaintenance,
		// The next periodic run retries anyway.
		MaxAttempts: 1,
	}
}
