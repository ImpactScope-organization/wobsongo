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

    const result = await callGoExtract({ url: pending.url });
    savePendingJob(result.jobId, {
      jid: pending.jid,
      waitingMessageId: pending.waitingMessageId,
      url: pending.url,
    });
  } catch (err) {
    console.error('[extract-callback] failed re-fetch /extract:', err);
    await conversationService.sendMessage(pending.jid, {
      text: '❌ An error occurred while fetching the transcription result.',
    });
    deletePendingJob(jobId);
  }
}
