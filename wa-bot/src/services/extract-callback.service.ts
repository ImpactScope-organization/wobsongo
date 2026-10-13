import {
  getPendingJob,
  deletePendingJob,
  savePendingJob,
  markAwaitingVoice,
} from './pending-job.store.js';
import { callGoExtract } from './go-client.service.js';
import * as conversationService from './conversation.service.js';
import * as telegramService from './telegram.service.js';
import { requestVoiceReply, deliverVoiceReply } from './voice-reply.service.js';
import { env } from '../config/env.js';
import type { ExtractCallbackStatus, ExtractData } from '../types/types.js';

const GENERIC_ERROR = '❌ Une erreur est survenue. Réessaie plus tard.';

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
      const text = (data.answer ?? data.transcript ?? '').trim();
      if (!text) {
        throw new Error(`empty answer from backend for jobId=${jobId}`);
      }

      if (pending.inputType === 'voice') {
        // The pending job stays alive until /callback/tts-done (or the fallback timer).
        await startVoiceReply(jobId, pending.platform, text);
        return;
      }

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
      inputType: pending.inputType,
    });
  } catch (err) {
    console.error('[extract-callback] failed to notify user or re-fetch result:', err);
    try {
      await sendToUser(pending.platform, pending.jid, GENERIC_ERROR);
    } catch (sendErr) {
      console.error('[extract-callback] failed to send error message:', sendErr);
    } finally {
      deletePendingJob(jobId);
    }
  }
}

// startVoiceReply asks the backend to synthesize the answer in the background.
async function startVoiceReply(
  jobId: string,
  platform: 'whatsapp' | 'telegram',
  text: string
): Promise<void> {
  // Store the fallback and extend the job before asking.
  markAwaitingVoice(jobId, text, env.voiceReplyWaitMs + 60_000);

  const requested = await requestVoiceReply(jobId, platform, text);
  if (!requested) {
    await handleTtsDone(jobId, 'failed', Buffer.alloc(0));
    return;
  }

  setTimeout(() => {
    handleTtsDone(jobId, 'failed', Buffer.alloc(0)).catch((err) => {
      console.error('[extract-callback] voice fallback timer failed:', err);
    });
  }, env.voiceReplyWaitMs).unref();
}

// handleTtsDone finishes a voice reply: it sends the audio when synthesis
// succeeded, and the text answer otherwise.
export async function handleTtsDone(
  jobId: string,
  status: 'completed' | 'failed',
  audio: Buffer
): Promise<void> {
  const pending = getPendingJob(jobId);
  if (!pending) {
    return;
  }

  // Claim the job first so the callback and the fallback timer never both send.
  deletePendingJob(jobId);

  if (status === 'completed' && audio.length > 0) {
    try {
      await deliverVoiceReply(pending.platform, pending.jid, audio);
      return;
    } catch (err) {
      console.error('[extract-callback] failed to deliver voice reply, sending text:', err);
    }
  }

  const fallback = pending.fallbackText?.trim();
  if (fallback) {
    await sendToUser(pending.platform, pending.jid, fallback);
  }
}
