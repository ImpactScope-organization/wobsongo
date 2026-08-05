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
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
)

const transcriptionJobTimeout = 5 * time.Minute

// workerComponentTranscription is this worker's name for notifyBotFailed's log prefix.
const workerComponentTranscription = "TranscriptionWorker"

// TranscriptionWorker processes transcription jobs by sending media URLs to the
// Modal ASR service and storing the resulting transcript in the database.
type TranscriptionWorker struct {
	river.WorkerDefaults[queue.TranscriptionJob]
	videoService        *service.VideoService
	modalURL            string
	asrModel            string
	asrSourceLang       string
	httpClient          data.HTTPClient
	botClient           *external.BotClient
	conversationService *service.ConversationService
}

// NewTranscriptionWorker creates a new TranscriptionWorker instance.
func NewTranscriptionWorker(
	videoService *service.VideoService,
	modalURL string,
	asrModel string,
	asrSourceLang string,
	botClient *external.BotClient,
	conversationService *service.ConversationService,
) *TranscriptionWorker {
	return &TranscriptionWorker{
		videoService:  videoService,
		modalURL:      modalURL,
		asrModel:      asrModel,
		asrSourceLang: asrSourceLang,
		httpClient: &http.Client{
			Timeout: transcriptionJobTimeout,
		},
		botClient:           botClient,
		conversationService: conversationService,
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

	internal.NotifyBotProgress(
		ctx,
		w.botClient,
		workerComponentTranscription,
		job.Args.ExtractionID,
		"📝 The video is being transcribed....",
	)

	// Load the Modal API endpoint from the environment.
	if w.modalURL == "" {
		err := errors.New("transcription worker: modalURL is not configured")
		internal.NotifyBotFailed(
			ctx,
			w.botClient,
			workerComponentTranscription,
			job.Args.ExtractionID,
			err,
		)
		return err
	}

	// Build the request payload for the Modal ASR service
	modalResp, err := w.callModalASR(ctx, job.Args.DownloadURL)
	if err != nil {
		internal.NotifyBotFailed(
			ctx,
			w.botClient,
			workerComponentTranscription,
			job.Args.ExtractionID,
			err,
		)
		return err
	}

	if err := w.videoService.UpdateVideoTranscription(
		ctx,
		pgtype.Text{String: modalResp.Transcript, Valid: modalResp.Transcript != ""},
		job.Args.VideoID,
	); err != nil {
		err = fmt.Errorf("failed to save transcription to db: %w", err)
		internal.NotifyBotFailed(
			ctx,
			w.botClient,
			workerComponentTranscription,
			job.Args.ExtractionID,
			err,
		)
		return err
	}

	log.Printf(
		"[TranscriptionWorker] Successfully processed VideoID %s | Language: %s",
		job.Args.VideoID, modalResp.LanguageDetected,
	)

	return w.dispatchFollowUp(ctx, job.Args, modalResp.Transcript)
}

// modalASRResponse is the decoded response body from the Modal ASR service.
type modalASRResponse struct {
	Transcript       string `json:"transcript"`
	LanguageDetected string `json:"language_detected"`
	Error            string `json:"error"`
}

// callModalASR builds and sends the transcription request to the Modal ASR
// service and returns the parsed response.
func (w *TranscriptionWorker) callModalASR(
	ctx context.Context,
	downloadURL string,
) (*modalASRResponse, error) {
	payload := map[string]string{
		"model":        w.asrModel,
		"audio_url":    downloadURL,
		"source_lang":  w.asrSourceLang,
		"audio_format": "mp4",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.modalURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute modal request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("modal API returned status %d", resp.StatusCode)
	}

	var modalResp modalASRResponse
	if err := json.NewDecoder(resp.Body).Decode(&modalResp); err != nil {
		return nil, fmt.Errorf("failed to decode modal response: %w", err)
	}
	if modalResp.Error != "" {
		return nil, fmt.Errorf("modal application error: %s", modalResp.Error)
	}

	return &modalResp, nil
}

// dispatchFollowUp enqueues the appropriate next step (agent turn, claim
// check, or a direct bot notification) once a transcript is ready.
func (w *TranscriptionWorker) dispatchFollowUp(
	ctx context.Context,
	args queue.TranscriptionJob,
	transcript string,
) error {
	switch {
	case args.ViaAgent:
		note := "The video transcript is ready:\n\n" + transcript
		if err := w.conversationService.EnqueueAgentTurn(ctx, queue.AgentTurnJob{
			Jid: args.Jid, ExtractionID: args.ExtractionID, SystemNote: note,
		}); err != nil {
			err = fmt.Errorf("failed to enqueue agent turn continuation: %w", err)
			internal.NotifyBotFailed(
				ctx,
				w.botClient,
				workerComponentTranscription,
				args.ExtractionID,
				err,
			)
			return err
		}

	case args.Jid != "":
		if err := w.videoService.EnqueueClaimCheckJob(ctx, queue.ClaimCheckJob{
			ExtractionID: args.ExtractionID,
			Text:         transcript,
			Jid:          args.Jid,
		}); err != nil {
			err = fmt.Errorf("failed to enqueue claim check: %w", err)
			internal.NotifyBotFailed(
				ctx,
				w.botClient,
				workerComponentTranscription,
				args.ExtractionID,
				err,
			)
			return err
		}

	default:
		if notifyErr := w.botClient.NotifyExtractDone(
			ctx, args.ExtractionID, "completed", "", nil,
		); notifyErr != nil {
			log.Printf("[TranscriptionWorker] Failed to notify the bot: %v", notifyErr)
		}
	}

	return nil
}
