package worker

import (
	"context"
	"fmt"
	"log"
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

// videoAudioFormat is the format hint sent to the ASR service for the audio
// downloaded from an extracted video.
const videoAudioFormat = "mp4"

// workerComponentTranscription is this worker's name for notifyBotFailed's log prefix.
const workerComponentTranscription = "TranscriptionWorker"

// transcriptionFailureMessage is the fixed French message shown to the user
// for any failure in this worker — the real cause is logged server-side by
// NotifyBotFailed, never forwarded to the user.
const transcriptionFailureMessage = "Une erreur est survenue pendant la transcription de la vidéo. Réessaie plus tard."

// TranscriptionWorker processes transcription jobs by sending media URLs to the
// ASR service and storing the resulting transcript in the database.
type TranscriptionWorker struct {
	river.WorkerDefaults[queue.TranscriptionJob]
	videoService        *service.VideoService
	transcriber         data.SpeechTranscriber
	botClient           *external.BotClient
	conversationService *service.ConversationService
}

// NewTranscriptionWorker creates a new TranscriptionWorker instance.
func NewTranscriptionWorker(
	videoService *service.VideoService,
	transcriber data.SpeechTranscriber,
	botClient *external.BotClient,
	conversationService *service.ConversationService,
) *TranscriptionWorker {
	return &TranscriptionWorker{
		videoService:        videoService,
		transcriber:         transcriber,
		botClient:           botClient,
		conversationService: conversationService,
	}
}

func (w *TranscriptionWorker) Timeout(_ *river.Job[queue.TranscriptionJob]) time.Duration {
	return transcriptionJobTimeout
}

// Work processes a transcription job by invoking the ASR service and
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
		internal.MsgTranscribing,
	)

	transcript, err := w.transcriber.TranscribeURL(ctx, job.Args.DownloadURL, videoAudioFormat)
	if err != nil {
		internal.NotifyBotFailed(
			ctx,
			w.botClient,
			workerComponentTranscription,
			job.Args.ExtractionID,
			err,
			transcriptionFailureMessage,
		)
		return err
	}

	if err := w.videoService.UpdateVideoTranscription(
		ctx,
		pgtype.Text{String: transcript, Valid: transcript != ""},
		job.Args.VideoID,
	); err != nil {
		err = fmt.Errorf("failed to save transcription to db: %w", err)
		internal.NotifyBotFailed(
			ctx,
			w.botClient,
			workerComponentTranscription,
			job.Args.ExtractionID,
			err,
			transcriptionFailureMessage,
		)
		return err
	}

	log.Printf("[TranscriptionWorker] Successfully processed VideoID %s", job.Args.VideoID)

	return w.dispatchFollowUp(ctx, job.Args, transcript)
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
			JID: args.Jid, ExtractionID: args.ExtractionID, SystemNote: note,
		}); err != nil {
			err = fmt.Errorf("failed to enqueue agent turn continuation: %w", err)
			internal.NotifyBotFailed(
				ctx,
				w.botClient,
				workerComponentTranscription,
				args.ExtractionID,
				err,
				transcriptionFailureMessage,
			)
			return err
		}

	case args.Jid != "":
		if err := w.videoService.EnqueueClaimCheckJob(ctx, queue.ClaimCheckJob{
			ExtractionID: args.ExtractionID,
			Text:         transcript,
			JID:          args.Jid,
		}); err != nil {
			err = fmt.Errorf("failed to enqueue claim check: %w", err)
			internal.NotifyBotFailed(
				ctx,
				w.botClient,
				workerComponentTranscription,
				args.ExtractionID,
				err,
				transcriptionFailureMessage,
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
