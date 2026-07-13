package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

const (
	apifyEventRunSucceeded = "ACTOR.RUN.SUCCEEDED"
	apifyStatusSucceeded   = "SUCCEEDED"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type ApifyService struct {
	apifyRepo    data.ApifyRepoer
	videoService *VideoService
	httpClient   HTTPClient
	apifyToken   string
}

// NewApifyService creates a new ApifyService.
func NewApifyService(
	apifyRepo data.ApifyRepoer,
	videoService *VideoService,
	httpClient HTTPClient,
	apifyToken string,
) *ApifyService {
	return &ApifyService{
		apifyRepo:    apifyRepo,
		videoService: videoService,
		httpClient:   httpClient,
		apifyToken:   apifyToken,
	}
}

// TriggerExtraction triggers a media extraction job by enqueuing it in the Apify repository.
func (s *ApifyService) TriggerExtraction(ctx context.Context, req *dto.ExtractionRequest) error {
	args := queue.ExtractMediaDTO{
		TargetURL:  req.TargetURL,
		WebhookURL: req.WebhookURL,
	}
	return s.apifyRepo.EnqueueExtraction(ctx, args)
}

// ProcessWebhook processes the validation logic from the Apify webhook response.
// It returns the datasetID if successful, or an empty string if ignored.
func (s *ApifyService) ProcessWebhook(
	ctx context.Context,
	payload *dto.ApifyWebhookPayload,
) (string, error) {
	if payload.EventType != apifyEventRunSucceeded ||
		payload.Resource.Status != apifyStatusSucceeded {
		log.Printf(
			"[ApifyWebhook] ignored event=%s status=%s",
			payload.EventType,
			payload.Resource.Status,
		)
		return "", nil
	}

	datasetID := payload.Resource.DefaultDatasetId

	// 1. Fetch dataset from Apify
	items, err := s.FetchDataset(ctx, datasetID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch dataset: %w", err)
	}

	// 2. Process and save items to the database
	if err := s.videoService.ProcessAndSaveApifyItems(ctx, items); err != nil {
		return "", fmt.Errorf("failed to save items to database: %w", err)
	}

	return datasetID, nil
}

// FetchDataset fetches the dataset from Apify using the provided datasetID.
func (s *ApifyService) FetchDataset(
	ctx context.Context,
	datasetID string,
) ([]dto.ApifyTikTokItem, error) {
	if s.apifyToken == "" {
		return nil, errors.New("apify service: apifyToken is not configured")
	}

	safeDatasetID := url.PathEscape(datasetID)
	reqURL := fmt.Sprintf("https://api.apify.com/v2/datasets/%s/items", safeDatasetID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apifyToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dataset: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apify API returned status: %d", resp.StatusCode)
	}

	var items []dto.ApifyTikTokItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode dataset json: %w", err)
	}

	return items, nil
}
