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
    // errorMsg is now always a complete, friendly French sentence supplied by
    // the Go side (internal.NotifyBotFailed) — never raw internal error text.
    await conversationService.sendMessage(pending.jid, {
      text: `❌ ${errorMsg || 'Une erreur est survenue. Réessaie plus tard.'}`.trim(),
    });
    deletePendingJob(jobId);
    return;
  }

   if (status === 'processing') {
    if (data?.answer) {
      await conversationService.sendMessage(pending.jid, { text: data.answer });
    }
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
      text: '❌ Une erreur est survenue lors de la récupération du résultat de la transcription. Réessaie plus tard.',
    });
    deletePendingJob(jobId);
  }
}
