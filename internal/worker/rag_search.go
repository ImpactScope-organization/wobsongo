package worker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

// ragSearchJobTimeout bounds the execution time of a single RAG search job.
const ragSearchJobTimeout = 60 * time.Second

// ragSearchAnswerLimit is the maximum number of RAG results used to
// generate the answer summary.
const ragSearchAnswerLimit = 5

// RAGSearchWorker performs a RAG search in the background and sends the
// generated answer to the bot via the extraction callback. The answer is
// generated on demand and is not persisted.
type RAGSearchWorker struct {
	river.WorkerDefaults[queue.RAGSearchJob]
	ragService *service.RAGService
	botClient  *external.BotClient
}

// NewRAGSearchWorker creates a new RAGSearchWorker.
func NewRAGSearchWorker(
	ragService *service.RAGService,
	botClient *external.BotClient,
) *RAGSearchWorker {
	return &RAGSearchWorker{
		ragService: ragService,
		botClient:  botClient,
	}
}

func (w *RAGSearchWorker) Timeout(_ *river.Job[queue.RAGSearchJob]) time.Duration {
	return ragSearchJobTimeout
}

// Work performs a RAG search for the transcript and notifies the bot with
// the generated answer.
func (w *RAGSearchWorker) Work(
	ctx context.Context,
	job *river.Job[queue.RAGSearchJob],
) error {
	log.Printf("[RAGSearchWorker] Starting RAG search for ExtractionID: %s", job.Args.ExtractionID)

	results, err := w.ragService.Search(ctx, job.Args.Transcript, ragSearchAnswerLimit)
	if err != nil {
		err = fmt.Errorf("RAG search failed: %w", err)
		w.notifyFailed(ctx, job.Args.ExtractionID, err)
		return err
	}

	answer := formatRAGSummary(results)

	log.Printf("[RAGSearchWorker] Successfully processed ExtractionID %s", job.Args.ExtractionID)

	if notifyErr := w.botClient.NotifyExtractDone(
		ctx,
		job.Args.ExtractionID,
		"completed",
		"",
		&external.ExtractCallbackData{
			Transcript: job.Args.Transcript,
			Answer:     answer,
		},
	); notifyErr != nil {
		log.Printf("[RAGSearchWorker] Failed to notify bot (answer will be lost): %v", notifyErr)
	}

	return nil
}

// notifyFailed notifies the bot that the RAG search job has failed.
func (w *RAGSearchWorker) notifyFailed(ctx context.Context, extractionID string, cause error) {
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
		log.Printf("[RAGSearchWorker] failed to notify bot (failed case): %v", err)
	}
}

// formatRAGSummary formats the top RAG search results into a numbered
// summary suitable for the bot response.
func formatRAGSummary(results []service.RAGResult) string {
	if len(results) == 0 {
		return "No relevant information was found in the knowledge database."
	}

	var b strings.Builder
	for i, r := range results {
		text := r.Text
		if r.Source == "fact" && r.ChunkText != "" {
			text = r.ChunkText
		}
		fmt.Fprintf(&b, "%d. %s\n\n", i+1, text)
	}
	return strings.TrimSpace(b.String())
}
