package internal

import (
	"context"
	"log"

	"github.com/impactscope-organization/wobsongo/external"
)

// User-facing progress messages shown while a job is running — French,
// informal "tu" register (WobSongo_Response_Logic_Heuristics_v2.md §5).
const (
	MsgCheckingClaim = "🔍 Je vérifie l'information..."
	MsgThinking      = "💬 Je réfléchis..."
	MsgCheckingVideo = "⏳ Un instant, je vérifie ça pour toi..."
	MsgTranscribing  = "📝 La vidéo est en cours de transcription..."
)

// notifyBotFailed sends a failed NotifyExtractDone callback to the bot.
// cause is logged server-side only — never forwarded to the user, since it's
// raw internal error text (often English, sometimes technical detail).
// userMessage is the French, user-facing sentence actually sent over the
// wire. Missing extraction IDs and callback failures are logged but not
// returned.
func NotifyBotFailed(
	ctx context.Context,
	botClient *external.BotClient,
	component string,
	extractionID string,
	cause error,
	userMessage string,
) {
	log.Printf("[%s] failure: %v", component, cause)
	if extractionID == "" {
		return
	}
	if err := botClient.NotifyExtractDone(
		ctx,
		extractionID,
		"failed",
		userMessage,
		nil,
	); err != nil {
		log.Printf("[%s] failed to notify bot (fait-doneled case): %v", component, err)
	}
}

// notifyBotProgress sends an interim status update to the bot.
func NotifyBotProgress(
	ctx context.Context,
	botClient *external.BotClient,
	component string,
	extractionID string,
	message string,
) {
	if extractionID == "" {
		return
	}
	if err := botClient.NotifyExtractDone(
		ctx,
		extractionID,
		"processing",
		"",
		&external.ExtractCallbackData{Answer: message},
	); err != nil {
		log.Printf("[%s] failed to notify progress: %v", component, err)
	}
}
