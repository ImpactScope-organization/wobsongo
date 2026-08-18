import { getPendingJob, deletePendingJob, savePendingJob } from './pending-job.store.js';
import { callGoExtract } from './go-client.service.js';
import * as conversationService from './conversation.service.js';
import * as telegramService from './telegram.service.js';
import type { ExtractCallbackStatus, ExtractData } from '../types/types.js';

async function sendToUser(platform: 'whatsapp' | 'telegram', jid: string, text: string) {
  if (platform === 'telegram') {
    await telegramService.sendMessage(jid, text);
  } else {
    await conversationService.sendMessage(jid, { text });
  }
}

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
    const text = `❌ ${errorMsg || 'Une erreur est survenue. Réessaie plus tard.'}`.trim();
    await sendToUser(pending.platform, pending.jid, text);
    deletePendingJob(jobId);
    return;
  }

  if (status === 'processing') {
    if (data?.answer) {
      await sendToUser(pending.platform, pending.jid, data.answer);
    }
    return;
  }

  try {
    if (data) {
      const text = data.answer ?? data.transcript ?? '';
      await sendToUser(pending.platform, pending.jid, text);
      deletePendingJob(jobId);
      return;
    }

    const result = await callGoExtract({ url: pending.url });
    savePendingJob(result.jobId, {
      jid: pending.jid,
      waitingMessageId: pending.waitingMessageId,
      url: pending.url,
      platform: pending.platform,
    });
  } catch (err) {
    console.error('[extract-callback] failed to notify user or re-fetch result:', err);
    await sendToUser(
      pending.platform,
      pending.jid,
      '❌ Une erreur est survenue lors de la récupération du résultat de la transcription. Réessaie plus tard.'
    );
    deletePendingJob(jobId);
  }
}
