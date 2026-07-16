package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

const (
	// apifyEventRunSucceeded represents the expected webhook event type for a successfully completed actor run.
	apifyEventRunSucceeded = "ACTOR.RUN.SUCCEEDED"

	// apifyStatusSucceeded represents the expected resource status for a successfully completed actor run.
	apifyStatusSucceeded = "SUCCEEDED"
)

type ApifyService struct {
	apifyRepo      data.ApifyRepoer
	videoRepo      data.VideoRepoer
	videoService   *VideoService
	httpClient     data.HTTPClient
	apifyToken     string
	baseWebhookURL string
}

// NewApifyService creates a new ApifyService.
func NewApifyService(
	apifyRepo data.ApifyRepoer,
	videoRepo data.VideoRepoer,
	videoService *VideoService,
	httpClient data.HTTPClient,
	apifyToken string,
	baseWebhookURL string,
) *ApifyService {
	return &ApifyService{
		apifyRepo:      apifyRepo,
		videoRepo:      videoRepo,
		videoService:   videoService,
		httpClient:     httpClient,
		apifyToken:     apifyToken,
		baseWebhookURL: baseWebhookURL,
	}
}

// TriggerExtraction is the bot's entry point. The bot checks the cache first,
// and only constructs an ExtractionRequest when the data is not found in the database.
func (s *ApifyService) TriggerExtraction(
	ctx context.Context,
	targetURL string,
) (*dto.ExtractResponse, error) {
	// 1. Cache check
	video, err := s.videoRepo.GetByVideoURL(ctx, targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing video: %w", err)
	}
	// If a valid cache is found, return the completed status immediately.
	if video != nil && video.TranscriptionText != nil && *video.TranscriptionText != "" {
		return &dto.ExtractResponse{
			Status: dto.StatusCompleted,
			JobID:  video.ID.String(),
			Data: &dto.ExtractData{
				Transcript: *video.TranscriptionText,
			},
		}, nil
	}

	// 2. If Cache miss generate a new extraction ID and construct the ExtractionRequest.
	extractionID := uuid.New().String()
	webhookURL := fmt.Sprintf(
		"%s/api/webhooks/apify?extractionId=%s",
		strings.TrimSuffix(s.baseWebhookURL, "/"),
		extractionID,
	)

	internalReq := &dto.ExtractionRequest{
		TargetURL:  targetURL,
		WebhookURL: webhookURL,
	}

	// Enqueue the job for background processing.
	args := queue.ExtractMediaDTO{
		ExtractionID: extractionID,
		TargetURL:    internalReq.TargetURL,
		WebhookURL:   internalReq.WebhookURL,
	}

	if err := s.apifyRepo.EnqueueExtraction(ctx, args); err != nil {
		return nil, fmt.Errorf("failed to enqueue extraction: %w", err)
	}

	return &dto.ExtractResponse{
		Status: dto.StatusProcessing,
		JobID:  extractionID,
	}, nil
}

// ProcessWebhook processes the validation logic from the Apify webhook response.
// It returns the datasetID if successful, or an empty string if ignored.
func (s *ApifyService) ProcessWebhook(
	ctx context.Context,
	payload *dto.ApifyWebhookPayload,
	extractionID string,
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
	if err := s.videoService.ProcessAndSaveApifyItems(
		ctx,
		items,
		extractionID,
	); err != nil {
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
