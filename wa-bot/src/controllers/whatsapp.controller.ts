import type { Request, Response } from 'express';
import { callGoAgentInbound } from '../services/go-client.service.js';
import { savePendingJob } from '../services/pending-job.store.js';
import * as conversationService from '../services/conversation.service.js';
import { SendMessageRequest } from '../types/types.js';

interface TwilioInboundBody {
  From?: string;
  Body?: string;
  [key: string]: unknown;
}

export async function handleIncomingMessage(req: Request, res: Response): Promise<void> {
  res.set('Content-Type', 'text/xml');
  res.send('<Response></Response>');

  const body = req.body as TwilioInboundBody;
  const rawFrom = body.From;
  const text = body.Body;

  if (!rawFrom || !text) return;

  const jid = rawFrom.replace('whatsapp:+', '');

  try {
    const result = await callGoAgentInbound({ jid, text: text.trim() });

    if (result.status === 'rejected') {
      await safeSendMessage(jid, {
        text: result.message ?? 'Désolé, je ne peux pas traiter cette demande.',
      });
      return;
    }

    savePendingJob(result.jobId, { jid, waitingMessageId: '', url: '' });
  } catch (err) {
    console.error('[whatsapp.controller] failed to process inbound message:', err);
    await safeSendMessage(jid, { text: '❌ Une erreur est survenue. Réessaie plus tard.' });
  }
}

async function safeSendMessage(jid: string, body: SendMessageRequest): Promise<void> {
  try {
    await conversationService.sendMessage(jid, body);
  } catch (err) {
    console.error('[whatsapp.controller] fallback sendMessage failed:', err);
  }
}
