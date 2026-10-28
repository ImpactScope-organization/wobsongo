package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	// voicePresignTTLSeconds is how long the ASR service has to fetch the audio.
	voicePresignTTLSeconds int64 = 600

	// voiceCleanupTimeout bounds the S3 delete after the job is done.
	voiceCleanupTimeout = 10 * time.Second

	workerComponentVoiceTranscription = "VoiceTranscriptionWorker"

	// Fixed French messages shown to the user; the real cause is only logged.
	voiceFailureMessage  = "Impossible de traiter le message vocal. Réessaie plus tard."
	voiceUnclearMessage  = "Désolé, je n'ai pas pu comprendre ce message vocal. Peux-tu le renvoyer ou l'écrire ?"
	voiceTooLargeMessage = "Ce message vocal est trop long. Envoie un message plus court ou écris-moi."
)

// VoiceTranscriptionWorker downloads an inbound voice message, has the ASR
// service transcribe it from a presigned S3 URL and starts an agent turn with
// the transcript.
type VoiceTranscriptionWorker struct {
	river.WorkerDefaults[queue.VoiceTranscriptionJob]
	downloader          data.VoiceMediaDownloader
	store               data.VoiceObjectStore
	presigner           data.ObjectPresigner
	transcriber         data.SpeechTranscriber
	conversationService *service.ConversationService
	botClient           *external.BotClient
}

// NewVoiceTranscriptionWorker creates a new VoiceTranscriptionWorker.
func NewVoiceTranscriptionWorker(
	downloader data.VoiceMediaDownloader,
	store data.VoiceObjectStore,
	presigner data.ObjectPresigner,
	transcriber data.SpeechTranscriber,
	conversationService *service.ConversationService,
	botClient *external.BotClient,
) *VoiceTranscriptionWorker {
	return &VoiceTranscriptionWorker{
		downloader:          downloader,
		store:               store,
		presigner:           presigner,
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

// Work downloads the audio, stores it in S3 for the ASR service, transcribes
// it, stores the transcript as the user's message and enqueues the agent turn
// that produces the answer.
func (w *VoiceTranscriptionWorker) Work(
	ctx context.Context,
	job *river.Job[queue.VoiceTranscriptionJob],
) error {
	args := job.Args
	log.Printf("[VoiceTranscriptionWorker] Starting for ExtractionID=%s", args.ExtractionID)

	format, ok := data.VoiceFormatFromMime(args.MimeType)
	if !ok {
		err := fmt.Errorf("unsupported voice mime type %q", args.MimeType)
		w.fail(ctx, args.ExtractionID, err, voiceFailureMessage)
		return river.JobCancel(err)
	}

	audio, err := w.downloader.Download(ctx, data.VoiceChannel(args.Channel), args.MediaRef)
	if err != nil {
		err = fmt.Errorf("failed to download voice message: %w", err)
		w.fail(ctx, args.ExtractionID, err, downloadFailureMessage(err))
		return err
	}

	key := voiceObjectKey(args.ExtractionID, format)
	if err := w.store.PutObject(
		ctx, key, bytes.NewReader(audio), int64(len(audio)), voiceContentType(format),
	); err != nil {
		err = fmt.Errorf("failed to store voice message: %w", err)
		w.fail(ctx, args.ExtractionID, err, voiceFailureMessage)
		return err
	}
	defer w.deleteObject(ctx, key)

	audioURL, err := w.presigner.GetPresignedGETURL(ctx, key, voicePresignTTLSeconds)
	if err != nil {
		err = fmt.Errorf("failed to presign voice message: %w", err)
		w.fail(ctx, args.ExtractionID, err, voiceFailureMessage)
		return err
	}

	transcript, err := w.transcriber.TranscribeURL(ctx, audioURL, format)
	if err != nil {
		w.fail(ctx, args.ExtractionID, err, voiceFailureMessage)
		return err
	}
	if transcript == "" {
		w.fail(ctx, args.ExtractionID, errors.New("empty transcript"), voiceUnclearMessage)
		return nil
	}

	messageID, err := w.conversationService.AppendMessageWithID(
		ctx, args.JID, model.ConversationRoleUser, transcript,
	)
	if err != nil {
		err = fmt.Errorf("failed to store voice transcript: %w", err)
		w.fail(ctx, args.ExtractionID, err, voiceFailureMessage)
		return err
	}

	if err := w.conversationService.EnqueueAgentTurn(ctx, queue.AgentTurnJob{
		JID:          args.JID,
		ExtractionID: args.ExtractionID,
		UserText:     transcript,
		MessageID:    messageID.String(),
		Format:       string(model.ClaimFormatAudio),
	}); err != nil {
		err = fmt.Errorf("failed to enqueue agent turn: %w", err)
		w.fail(ctx, args.ExtractionID, err, voiceFailureMessage)
		return err
	}
	return nil
}

// deleteObject removes the temporary audio, detached from ctx so cleanup still
// runs when the job's own deadline has passed. A failure is only logged, the
// transcript is already done at this point.
func (w *VoiceTranscriptionWorker) deleteObject(ctx context.Context, key string) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), voiceCleanupTimeout)
	defer cancel()
	if err := w.store.DeleteObject(cleanupCtx, key); err != nil {
		log.Printf("[%s] failed to delete %s: %v", workerComponentVoiceTranscription, key, err)
	}
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

// downloadFailureMessage picks the message shown to the user for a failed
// download: a too-long message gets a specific hint, everything else is generic.
func downloadFailureMessage(err error) string {
	switch {
	case errors.Is(err, data.ErrVoiceTooLarge):
		return voiceTooLargeMessage
	case errors.Is(err, data.ErrVoiceEmpty):
		return voiceUnclearMessage
	default:
		return voiceFailureMessage
	}
}

// voiceObjectKey returns the S3 key of a voice message. The file name is the
// SHA-256 of the extraction ID, which keeps it unique per job and matches the
// <sha256>.<ext> shape that presigning requires.
func voiceObjectKey(extractionID, format string) string {
	sum := sha256.Sum256([]byte(extractionID))
	return fmt.Sprintf(
		"%s%s.%s",
		data.ObjectPrefixForIntent(data.VoiceNoteUploadIntent),
		hex.EncodeToString(sum[:]),
		format,
	)
}

// voiceContentType maps an ASR format hint back to a Content-Type for S3.
func voiceContentType(format string) string {
	switch format {
	case "ogg":
		return "audio/ogg"
	case "mp3":
		return "audio/mpeg"
	case "m4a":
		return "audio/mp4"
	case "aac":
		return "audio/aac"
	case "amr":
		return "audio/amr"
	case "webm":
		return "audio/webm"
	case "wav":
		return "audio/wav"
	default:
		return "application/octet-stream"
	}
}
