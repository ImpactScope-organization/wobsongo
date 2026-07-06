package service

import (
	"context"
	"log"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

type ApifyService struct {
	repo data.ApifyRepoer
}

// NewApifyService creates a new ApifyService.
func NewApifyService(repo data.ApifyRepoer) *ApifyService {
	return &ApifyService{
		repo: repo,
	}
}

// TriggerExtraction triggers a media extraction job by enqueuing it in the Apify repository.
func (s *ApifyService) TriggerExtraction(ctx context.Context, req *dto.ExtractionRequest) error {
	args := queue.ExtractMediaDTO{
		TargetURL:  req.TargetURL,
		WebhookURL: req.WebhookURL,
	}
	return s.repo.EnqueueExtraction(ctx, args)
}

// ProcessWebhook processes the validation logic from the Apify webhook response.
// It returns the datasetID if successful, or an empty string if ignored.
func (s *ApifyService) ProcessWebhook(
	payload *dto.ApifyWebhookPayload,
) (string, error) {
	if payload.EventType != "ACTOR.RUN.SUCCEEDED" || payload.Resource.Status != "SUCCEEDED" {
		log.Printf(
			"[ApifyWebhook] ignored event=%s status=%s",
			payload.EventType,
			payload.Resource.Status,
		)
		return "", nil
	}

	datasetID := payload.Resource.DefaultDatasetId

	return datasetID, nil
}
