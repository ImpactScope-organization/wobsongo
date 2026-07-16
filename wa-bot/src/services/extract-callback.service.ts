import { getPendingJob, deletePendingJob } from './pending-job.store.js';
import { callGoExtract } from './go-client.service.js';
import * as conversationService from './conversation.service.js';
import type { ExtractCallbackStatus } from '../types/types.js';

// handleExtractDone is triggered when the Go backend hits the POST /callback/extract-done endpoint.
export async function handleExtractDone(
  jobId: string,
  status: ExtractCallbackStatus,
  errorMsg?: string
): Promise<void> {
  const pending = getPendingJob(jobId);
  if (!pending) {
    console.warn(`[extract-callback] unknown jobId: ${jobId}`);
    return;
  }

  if (status === 'failed') {
    await conversationService.sendMessage(pending.jid, {
      text: `❌ Failed to process video. ${errorMsg ?? ''}`.trim(),
      replyToMessageId: pending.waitingMessageId,
    });
    deletePendingJob(jobId);
    return;
  }

  try {
    // Re-call the /extract endpoint using the SAME url. 
    const result = await callGoExtract(pending.url);

    if (result.status === 'completed' && result.data) {
      await conversationService.sendMessage(pending.jid, {
        text: `📝 Transcription result:\n\n${result.data.transcript}`,
        replyToMessageId: pending.waitingMessageId,
      });
    } else {
      await conversationService.sendMessage(pending.jid, {
        text: '⚠️ The video has been processed, but the result cannot be fetched yet. Please send the link again.',
        replyToMessageId: pending.waitingMessageId,
      });
    }
  } catch (err) {
    console.error('[extract-callback] gagal re-fetch /extract:', err);
    await conversationService.sendMessage(pending.jid, {
      text: '❌ An error occurred while fetching the transcription result.',
      replyToMessageId: pending.waitingMessageId,
    });
  } finally {
    deletePendingJob(jobId);
  }
}