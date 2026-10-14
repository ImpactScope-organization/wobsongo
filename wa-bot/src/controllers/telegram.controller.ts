import type { Request, Response } from 'express';
import { savePendingJob } from '../services/pending-job.store.js';
import * as telegramService from '../services/telegram.service.js';
import { VOICE_MESSAGES } from '../utils/voice-messages.js';
import { env } from '../config/env.js';
import { callGoAgentInbound, callGoAgentInboundVoice } from '../services/go-client.service.js';
import type { ExtractResponse, InputType } from '../types/types.js';

interface TelegramAudioPayload {
  file_id: string;
  duration?: number;
  mime_type?: string;
}

interface TelegramUpdate {
  message?: {
    text?: string;
    chat?: { id: number | string };
    voice?: TelegramAudioPayload;
    audio?: TelegramAudioPayload;
  };
}

export async function handleTelegramWebhook(req: Request, res: Response): Promise<void> {
  res.sendStatus(200);

  const message = (req.body as TelegramUpdate).message;
  const chatId = message?.chat?.id?.toString();
  const voice = message?.voice ?? message?.audio;

  if (!message || !chatId || (!message.text && !voice)) {
    console.warn('[telegram.controller] ignoring unsupported update:', JSON.stringify(req.body));
    return;
  }

  const inputType: InputType = voice ? 'voice' : 'text';
  const msgs = VOICE_MESSAGES[env.voiceReplyLangTelegram];

  try {
    let result: ExtractResponse;

    if (voice) {
      await telegramService.sendMessage(chatId, msgs.received);
      result = await callGoAgentInboundVoice({
        jid: chatId,
        channel: 'telegram',
        mediaRef: voice.file_id,
        mimeType: voice.mime_type ?? 'audio/ogg',
      });
    } else {
      result = await callGoAgentInbound({ jid: chatId, text: (message.text ?? '').trim() });
    }

    if (result.status === 'rejected') {
      await telegramService.sendMessage(
        chatId,
        result.message ?? 'Désolé, je ne peux pas traiter cette demande.'
      );
      return;
    }

    savePendingJob(result.jobId, {
      jid: chatId,
      waitingMessageId: '',
      url: '',
      platform: 'telegram',
      inputType,
    });
  } catch (err) {
    console.error('[telegram.controller] failed to process inbound message:', err);
    await telegramService.sendMessage(
      chatId,
      voice ? msgs.error : '❌ Une erreur est survenue. Réessaie plus tard.'
    );
  }
}
