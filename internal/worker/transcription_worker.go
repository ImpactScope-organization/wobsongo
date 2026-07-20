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

	"github.com/impactscope-organization/wobsongo/external"
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
	river.WorkerDefaults[queue.TranscriptionJob]
	videoService  *service.VideoService
	modalURL      string
	asrModel      string
	asrSourceLang string
	httpClient    data.HTTPClient
	botClient     *external.BotClient
}

// NewTranscriptionWorker creates a new TranscriptionWorker instance.
func NewTranscriptionWorker(
	videoService *service.VideoService,
	modalURL string,
	asrModel string,
	asrSourceLang string,
	botClient *external.BotClient,
) *TranscriptionWorker {
	return &TranscriptionWorker{
		videoService:  videoService,
		modalURL:      modalURL,
		asrModel:      asrModel,
		asrSourceLang: asrSourceLang,
		httpClient: &http.Client{
			Timeout: transcriptionJobTimeout,
		},
		botClient: botClient,
	}
}

func (w *TranscriptionWorker) Timeout(_ *river.Job[queue.TranscriptionJob]) time.Duration {
	return transcriptionJobTimeout
}

// Work processes a transcription job by invoking the Modal ASR API and
// persisting the transcription result.
func (w *TranscriptionWorker) Work(
	ctx context.Context,
	job *river.Job[queue.TranscriptionJob],
) error {
	log.Printf("[TranscriptionWorker] Starting transcription for VideoID: %s", job.Args.VideoID)

	// Load the Modal API endpoint from the environment.
	if w.modalURL == "" {
		err := errors.New("transcription worker: modalURL is not configured")
		w.notifyFailed(ctx, job.Args.ExtractionID, err)
		return err
	}

	// Build the request payload for the Modal ASR service
	payload := map[string]string{
		"model":        w.asrModel,
		"audio_url":    job.Args.DownloadURL,
		"source_lang":  w.asrSourceLang,
		"audio_format": "mp4",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		err = fmt.Errorf("failed to marshal modal payload: %w", err)
		w.notifyFailed(ctx, job.Args.ExtractionID, err)
		return err
	}

	// Send the transcription request to Modal.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.modalURL, bytes.NewBuffer(body))
	if err != nil {
		err = fmt.Errorf("failed to create http request: %w", err)
		w.notifyFailed(ctx, job.Args.ExtractionID, err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to execute modal request: %w", err)
		w.notifyFailed(ctx, job.Args.ExtractionID, err)
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("modal API returned status %d", resp.StatusCode)
		w.notifyFailed(ctx, job.Args.ExtractionID, err)
		return err
	}

	// Decode the transcription response from Modal.
	var modalResp struct {
		Transcript       string `json:"transcript"`
		LanguageDetected string `json:"language_detected"`
		Error            string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modalResp); err != nil {
		err = fmt.Errorf("failed to decode modal response: %w", err)
		w.notifyFailed(ctx, job.Args.ExtractionID, err)
		return err
	}

	if modalResp.Error != "" {
		err := fmt.Errorf("modal application error: %s", modalResp.Error)
		w.notifyFailed(ctx, job.Args.ExtractionID, err)
		return err
	}

	// Convert the transcript into pgtype.Text for database storage.
	dbText := pgtype.Text{
		String: modalResp.Transcript,
		Valid:  modalResp.Transcript != "",
	}

	// Persist the transcription result.
	err = w.videoService.UpdateVideoTranscription(ctx, dbText, job.Args.VideoID)
	if err != nil {
		err = fmt.Errorf("failed to save transcription to db: %w", err)
		w.notifyFailed(ctx, job.Args.ExtractionID, err)
		return err
	}

	log.Printf(
		"[TranscriptionWorker] Successfully processed VideoID %s | Language: %s",
		job.Args.VideoID,
		modalResp.LanguageDetected,
	)

	// Notify the external bot that this specific extraction job has successfully completed.
	if notifyErr := w.botClient.NotifyExtractDone(
		ctx,
		job.Args.ExtractionID,
		"completed",
		"",
		nil,
	); notifyErr != nil {
		log.Printf("[TranscriptionWorker] gagal notify bot: %v", notifyErr)
	}

	return nil
}

// notifyFailed is a helper to avoid writing repetitive if-err checks at every point of failure.
func (w *TranscriptionWorker) notifyFailed(ctx context.Context, extractionID string, cause error) {
	if extractionID == "" {
		return
	}
	if err := w.botClient.NotifyExtractDone(
		ctx,
		extractionID,
		"failed",
		cause.Error(),
		nil,
	); err != nil {
		log.Printf("[TranscriptionWorker] failed to notify bot (failed case): %v", err)
	}
}
