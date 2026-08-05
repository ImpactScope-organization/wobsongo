package internal

import (
	"context"
	"log"

	"github.com/impactscope-organization/wobsongo/external"
)

// notifyBotFailed sends a failed NotifyExtractDone callback to the bot.
// Missing extraction IDs and callback failures are logged but not returned.
func NotifyBotFailed(
	ctx context.Context,
	botClient *external.BotClient,
	component string,
	extractionID string,
	cause error,
) {
	if extractionID == "" {
		return
	}
	if err := botClient.NotifyExtractDone(
		ctx,
		extractionID,
		"failed",
		cause.Error(),
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
