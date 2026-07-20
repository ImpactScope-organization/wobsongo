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

	// ragSearchLimit bounds how many fused RAG results are considered when
	// summarizing a video's transcript for the bot -- same reasoning as
	// ragDefaultLimit in cmd/rag.go, just scoped to this use case.
	ragSearchLimit = 5

	// ragSummaryPointCount caps how many top results are included in the
	// summary sent to the user, so WhatsApp messages stay readable.
	ragSummaryPointCount = 3

	// ragSummaryCharLimit bounds each summarized point's length.
	ragSummaryCharLimit = 400
)

type ApifyService struct {
	apifyRepo      data.ApifyRepoer
	videoRepo      data.VideoRepoer
	videoService   *VideoService
	ragService     *RAGService
	httpClient     data.HTTPClient
	apifyToken     string
	baseWebhookURL string
}

// NewApifyService creates a new ApifyService.
func NewApifyService(
	apifyRepo data.ApifyRepoer,
	videoRepo data.VideoRepoer,
	videoService *VideoService,
	ragService *RAGService,
	httpClient data.HTTPClient,
	apifyToken string,
	baseWebhookURL string,
) *ApifyService {
	return &ApifyService{
		apifyRepo:      apifyRepo,
		videoRepo:      videoRepo,
		videoService:   videoService,
		ragService:     ragService,
		httpClient:     httpClient,
		apifyToken:     apifyToken,
		baseWebhookURL: baseWebhookURL,
	}
}

// TriggerExtraction is the bot's entry point. Checks the cache first,
// and only constructs an ExtractionRequest when the data is not found in the database.
func (s *ApifyService) TriggerExtraction(
	ctx context.Context,
	targetURL string,
) (*dto.ExtractResponse, error) {
	// 1. Cache check
	video, err := s.videoRepo.GetByVideoURL(ctx, targetURL)
	if err != nil && !errors.Is(err, data.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing video: %w", err)
	}

	// If a valid cache is found, reuse the existing transcript by enqueueing a RAG search
	// job and return a processing response. Duplicate RAG jobs for the same
	// extraction are ignored by the queue.
	if video != nil && video.TranscriptionText != nil && *video.TranscriptionText != "" {
		extractionID := video.ID.String()

		if err := s.videoRepo.EnqueueRAGSearchJob(ctx, queue.RAGSearchJob{
			ExtractionID: extractionID,
			Transcript:   *video.TranscriptionText,
		}); err != nil {
			return nil, fmt.Errorf("failed to enqueue rag search: %w", err)
		}

		return &dto.ExtractResponse{
			Status: dto.StatusProcessing,
			JobID:  extractionID,
		}, nil
	}

	// 2. If Cache miss generate a new extraction ID and construct the ExtractionRequest.
	extractionID := uuid.New().String()
	webhookURL := fmt.Sprintf(
		"%s/api/webhooks/apify?extractionId=%s",
		strings.TrimSuffix(s.baseWebhookURL, "/"),
		extractionID,
	)

	// Enqueue the job for background processing.
	args := queue.ExtractMediaDTO{
		ExtractionID: extractionID,
		TargetURL:    targetURL,
		WebhookURL:   webhookURL,
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

// formatRAGSummary formats the top RAG search results into a numbered
// summary for use in the answer-generation prompt.
func formatRAGSummary(results []RAGResult) string {
	if len(results) == 0 {
		return "No relevant information was found in the knowledge database."
	}

	var sb strings.Builder
	count := 0
	for _, r := range results {
		if count >= ragSummaryPointCount {
			break
		}
		text := r.Text
		if r.Source == "fact" && r.ChunkText != "" {
			text = r.ChunkText
		}
		count++
		sb.WriteString(fmt.Sprintf("%d. %s\n\n", count, truncateRAGText(text, ragSummaryCharLimit)))
	}
	return strings.TrimSpace(sb.String())
}

// truncateRAGText truncates a string to at most n runes, appending an
// ellipsis if the text exceeds the limit.
func truncateRAGText(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}