import { getPendingJob, deletePendingJob, savePendingJob } from './pending-job.store.js';
import { callGoExtract } from './go-client.service.js';
import * as conversationService from './conversation.service.js';
import type { ExtractCallbackStatus, ExtractData } from '../types/types.js';

// handleExtractDone is triggered when the Go backend hits the POST /callback/extract-done endpoint.
export async function handleExtractDone(
  jobId: string,
  status: ExtractCallbackStatus,
  errorMsg?: string,
  data?: ExtractData
): Promise<void> {
  const pending = getPendingJob(jobId);
  if (!pending) {
    console.warn(`[extract-callback] unknown jobId: ${jobId}`);
    return;
  }

  if (status === 'failed') {
    await conversationService.sendMessage(pending.jid, {
      text: `❌ Failed to process video. ${errorMsg ?? ''}`.trim(),
    });
    deletePendingJob(jobId);
    return;
  }

  try {
    if (data) {
      // The callback already includes the final result.
      await conversationService.sendMessage(pending.jid, {
        text: data.answer ?? data.transcript ?? '',
      });
      deletePendingJob(jobId);
      return;
    }

    // Pull model fallback (transcription flow): re-call with the same URL.
    const result = await callGoExtract(pending.url);

    if (result.status === 'completed' && result.data) {
      await conversationService.sendMessage(pending.jid, {
        text: result.data.answer ?? result.data.transcript,
      });
      deletePendingJob(jobId);
    } else {
      // Cache-hit but RAG job just got (re-)enqueued under its OWN jobId
      // (result.jobId — the video's ID). Re-save the pendingJob under
      // that id so the eventual RAG completion can still find it.
      savePendingJob(result.jobId, {
        jid: pending.jid,
        waitingMessageId: pending.waitingMessageId,
        url: pending.url,
      });
    }
  } catch (err) {
    console.error('[extract-callback] gagal re-fetch /extract:', err);
    await conversationService.sendMessage(pending.jid, {
      text: '❌ An error occurred while fetching the transcription result.',
    });
    deletePendingJob(jobId);
  }
}