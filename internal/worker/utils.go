package worker

import (
	"context"
	"log"

	"github.com/impactscope-organization/wobsongo/external"
)

// notifyBotFailed sends a failed NotifyExtractDone callback to the bot.
// Missing extraction IDs and callback failures are logged but not returned.
func notifyBotFailed(
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
		log.Printf("[%s] failed to notify bot (failed case): %v", component, err)
	}
}
