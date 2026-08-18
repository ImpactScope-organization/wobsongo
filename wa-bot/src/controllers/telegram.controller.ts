import type { Request, Response } from 'express';
import { callGoAgentInbound } from '../services/go-client.service.js';
import { savePendingJob } from '../services/pending-job.store.js';
import * as telegramService from '../services/telegram.service.js';

interface TelegramUpdate {
  message?: {
    text?: string;
    chat?: { id: number | string };
  };
}

export async function handleTelegramWebhook(req: Request, res: Response): Promise<void> {
  res.sendStatus(200);

  const body = req.body as TelegramUpdate;
  const message = body.message;

  if (!message || !message.text || !message.chat?.id) {
    console.warn('[telegram.controller] ignoring non-text update:', JSON.stringify(req.body));
    return;
  }

  const chatId = message.chat.id.toString();
  const text = message.text;

  try {
    const result = await callGoAgentInbound({ jid: chatId, text: text.trim() });

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
    });
  } catch (err) {
    console.error('[telegram.controller] failed to process inbound message:', err);
    await telegramService.sendMessage(chatId, '❌ Une erreur est survenue. Réessaie plus tard.');
  }
}
