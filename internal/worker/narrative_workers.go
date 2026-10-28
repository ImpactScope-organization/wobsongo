package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

// clusterClaimJobTimeout bounds one cluster run (embedding + statement LLM call).
const clusterClaimJobTimeout = 3 * time.Minute

// ClusterClaimWorker groups a stored claim into a narrative.
type ClusterClaimWorker struct {
	river.WorkerDefaults[queue.ClusterClaimJob]
	narrativeService *service.NarrativeService
}

// NewClusterClaimWorker creates a new ClusterClaimWorker.
func NewClusterClaimWorker(narrativeService *service.NarrativeService) *ClusterClaimWorker {
	return &ClusterClaimWorker{narrativeService: narrativeService}
}

func (w *ClusterClaimWorker) Timeout(_ *river.Job[queue.ClusterClaimJob]) time.Duration {
	return clusterClaimJobTimeout
}

// Work groups the claim into a narrative.
func (w *ClusterClaimWorker) Work(
	ctx context.Context,
	job *river.Job[queue.ClusterClaimJob],
) error {
	claimID, err := uuid.Parse(job.Args.ClaimID)
	if err != nil {
		// A malformed id never becomes valid, so do not retry.
		return river.JobCancel(fmt.Errorf("invalid claim id %q: %w", job.Args.ClaimID, err))
	}
	if err := w.narrativeService.ClusterClaim(ctx, claimID); err != nil {
		return fmt.Errorf("failed to cluster claim %s: %w", claimID, err)
	}
	return nil
}

// DeactivateNarrativesWorker deactivates narratives without recent claims.
type DeactivateNarrativesWorker struct {
	river.WorkerDefaults[queue.DeactivateNarrativesJob]
	narrativeService *service.NarrativeService
	window           time.Duration
}

// NewDeactivateNarrativesWorker creates a new DeactivateNarrativesWorker.
func NewDeactivateNarrativesWorker(
	narrativeService *service.NarrativeService,
	window time.Duration,
) *DeactivateNarrativesWorker {
	return &DeactivateNarrativesWorker{narrativeService: narrativeService, window: window}
}

// Work deactivates every narrative without a claim inside the activity window.
func (w *DeactivateNarrativesWorker) Work(
	ctx context.Context,
	_ *river.Job[queue.DeactivateNarrativesJob],
) error {
	n, err := w.narrativeService.DeactivateStale(ctx, w.window)
	if err != nil {
		return fmt.Errorf("failed to deactivate narratives: %w", err)
	}
	log.Printf("[DeactivateNarrativesWorker] deactivated=%d window=%s", n, w.window)
	return nil
}
