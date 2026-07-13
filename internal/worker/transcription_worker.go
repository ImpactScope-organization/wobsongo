package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
)

const transcriptionJobTimeout = 5 * time.Minute

// TranscriptionWorker processes transcription jobs by sending media URLs to the
// Modal ASR service and storing the resulting transcript in the database.
type TranscriptionWorker struct {
	river.WorkerDefaults[queue.TranscriptionJobDTO]
	videoRepo  data.VideoRepoer
	modalURL   string
	httpClient service.HTTPClient
}

// NewTranscriptionWorker creates a new TranscriptionWorker instance.
func NewTranscriptionWorker(
	videoRepo data.VideoRepoer,
	modalURL string,
) *TranscriptionWorker {
	return &TranscriptionWorker{
		videoRepo: videoRepo,
		modalURL:  modalURL,
		httpClient: &http.Client{
			Timeout: transcriptionJobTimeout,
		},
	}
}

func (w *TranscriptionWorker) Timeout(_ *river.Job[queue.TranscriptionJobDTO]) time.Duration {
	return transcriptionJobTimeout
}

// Work processes a transcription job by invoking the Modal ASR API and
// persisting the transcription result.
func (w *TranscriptionWorker) Work(
	ctx context.Context,
	job *river.Job[queue.TranscriptionJobDTO],
) error {
	log.Printf("[TranscriptionWorker] Starting transcription for VideoID: %s", job.Args.VideoID)

	// Load the Modal API endpoint from the environment.
	if w.modalURL == "" {
		return errors.New("transcription worker: modalURL is not configured")
	}

	// Build the request payload for the Modal ASR service.
	payload := map[string]string{
		"model":        "Omnilingual ASR",
		"audio_url":    job.Args.DownloadURL,
		"source_lang":  "english",
		"audio_format": "mp4",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal modal payload: %w", err)
	}

	// Send the transcription request to Modal.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.modalURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute modal request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("modal API returned status %d", resp.StatusCode)
	}

	// Decode the transcription response from Modal.
	var modalResp struct {
		Transcript       string `json:"transcript"`
		LanguageDetected string `json:"language_detected"`
		Error            string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modalResp); err != nil {
		return fmt.Errorf("failed to decode modal response: %w", err)
	}

	if modalResp.Error != "" {
		return fmt.Errorf("modal application error: %s", modalResp.Error)
	}

	// Convert the transcript into pgtype.Text for database storage.
	dbText := pgtype.Text{
		String: modalResp.Transcript,
		Valid:  modalResp.Transcript != "",
	}

	// Persist the transcription result.
	err = w.videoRepo.UpdateVideoTranscription(ctx, dbText, job.Args.VideoID)
	if err != nil {
		return fmt.Errorf("failed to save transcription to db: %w", err)
	}

	log.Printf(
		"[TranscriptionWorker] Successfully processed VideoID %s | Language: %s",
		job.Args.VideoID,
		modalResp.LanguageDetected,
	)

	return nil
}
