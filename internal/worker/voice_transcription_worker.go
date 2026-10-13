package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

const (
	// voiceTranscriptionTimeout bounds one transcription job (5 minutes),
	// matching the ASR client's HTTP timeout.
	voiceTranscriptionTimeout = 5 * time.Minute

	workerComponentVoiceTranscription = "VoiceTranscriptionWorker"

	// Fixed French messages shown to the user; the real cause is only logged.
	voiceFailureMessage = "Impossible de traiter le message vocal. Réessaie plus tard."
	voiceUnclearMessage = "Désolé, je n'ai pas pu comprendre ce message vocal. Peux-tu le renvoyer ou l'écrire ?"
)

// VoiceTranscriptionWorker transcribes an inbound voice message and starts an
// agent turn with the transcript.
type VoiceTranscriptionWorker struct {
	river.WorkerDefaults[queue.VoiceTranscriptionJob]
	transcriber         data.SpeechTranscriber
	conversationService *service.ConversationService
	botClient           *external.BotClient
}

// NewVoiceTranscriptionWorker creates a new VoiceTranscriptionWorker.
func NewVoiceTranscriptionWorker(
	transcriber data.SpeechTranscriber,
	conversationService *service.ConversationService,
	botClient *external.BotClient,
) *VoiceTranscriptionWorker {
	return &VoiceTranscriptionWorker{
		transcriber:         transcriber,
		conversationService: conversationService,
		botClient:           botClient,
	}
}

func (w *VoiceTranscriptionWorker) Timeout(
	_ *river.Job[queue.VoiceTranscriptionJob],
) time.Duration {
	return voiceTranscriptionTimeout
}

// Work transcribes the audio, stores the transcript as the user's message and
// enqueues the agent turn that produces the answer.
func (w *VoiceTranscriptionWorker) Work(
	ctx context.Context,
	job *river.Job[queue.VoiceTranscriptionJob],
) error {
	args := job.Args
	log.Printf("[VoiceTranscriptionWorker] Starting for ExtractionID=%s", args.ExtractionID)

	transcript, err := w.transcriber.Transcribe(ctx, args.Audio, args.Format)
	if err != nil {
		w.fail(ctx, args.ExtractionID, err, voiceFailureMessage)
		return err
	}
	if transcript == "" {
		w.fail(ctx, args.ExtractionID, errors.New("empty transcript"), voiceUnclearMessage)
		return nil
	}

	if err := w.conversationService.AppendMessage(
		ctx, args.Jid, model.ConversationRoleUser, transcript,
	); err != nil {
		err = fmt.Errorf("failed to store voice transcript: %w", err)
		w.fail(ctx, args.ExtractionID, err, voiceFailureMessage)
		return err
	}

	if err := w.conversationService.EnqueueAgentTurn(ctx, queue.AgentTurnJob{
		Jid: args.Jid, ExtractionID: args.ExtractionID, UserText: transcript,
	}); err != nil {
		err = fmt.Errorf("failed to enqueue agent turn: %w", err)
		w.fail(ctx, args.ExtractionID, err, voiceFailureMessage)
		return err
	}
	return nil
}

// fail logs the cause and tells the bot, detached from ctx so the
// notification still goes out when the job's own deadline has passed.
func (w *VoiceTranscriptionWorker) fail(
	ctx context.Context,
	extractionID string,
	cause error,
	userMessage string,
) {
	internal.NotifyBotFailed(
		context.WithoutCancel(ctx),
		w.botClient,
		workerComponentVoiceTranscription,
		extractionID,
		cause,
		userMessage,
	)
}
