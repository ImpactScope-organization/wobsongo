import type { Request, Response } from 'express';
import { callGoAgentInbound } from '../services/go-client.service.js';
import { savePendingJob } from '../services/pending-job.store.js';
import * as conversationService from '../services/conversation.service.js';
import { downloadTwilioMedia } from '../services/audio-download.service.js';
import { transcribeAudio } from '../services/modal-asr.service.js';
import { VOICE_MESSAGES } from '../utils/voice-messages.js';
import { env } from '../config/env.js';
import { SendMessageRequest } from '../types/types.js';
import type { InputType } from '../types/types.js';

interface TwilioInboundBody {
  From?: string;
  Body?: string;
  NumMedia?: string;
  MediaUrl0?: string;
  MediaContentType0?: string;
  [key: string]: unknown;
}

export async function handleIncomingMessage(req: Request, res: Response): Promise<void> {
  res.set('Content-Type', 'text/xml');
  res.send('<Response></Response>');

  const body = req.body as TwilioInboundBody;
  const rawFrom = body.From;
  if (!rawFrom) {
    console.warn('[whatsapp.controller] ignoring payload without From:', JSON.stringify(body));
    return;
  }

  const mediaUrl = body.MediaUrl0;
  const mediaType = body.MediaContentType0 ?? '';
  const isVoice = Number(body.NumMedia ?? 0) > 0 && !!mediaUrl && mediaType.startsWith('audio/');

  if (!isVoice && !body.Body) {
    console.warn(
      '[whatsapp.controller] ignoring payload without Body/audio:',
      JSON.stringify(body)
    );
    return;
  }

  const jid = rawFrom.replace('whatsapp:+', '');
  const inputType: InputType = isVoice ? 'voice' : 'text';
  const msgs = VOICE_MESSAGES[env.voiceReplyLangWhatsapp];

  try {
    let text = (body.Body ?? '').trim();

    if (isVoice && mediaUrl) {
      await safeSendMessage(jid, { text: msgs.received });
      const audio = await downloadTwilioMedia(mediaUrl, mediaType);
      text = await transcribeAudio(audio.buffer, audio.format);
      if (!text) {
        await safeSendMessage(jid, { text: msgs.unclear });
        return;
      }
    }

    const result = await callGoAgentInbound({ jid, text });

    if (result.status === 'rejected') {
      await safeSendMessage(jid, {
        text: result.message ?? 'Désolé, je ne peux pas traiter cette demande.',
      });
      return;
    }

    savePendingJob(result.jobId, {
      jid,
      waitingMessageId: '',
      url: '',
      platform: 'whatsapp',
      inputType,
    });
  } catch (err) {
    console.error('[whatsapp.controller] failed to process inbound message:', err);
    await safeSendMessage(jid, {
      text: isVoice ? msgs.error : '❌ Une erreur est survenue. Réessaie plus tard.',
    });
  }
}

async function safeSendMessage(jid: string, body: SendMessageRequest): Promise<void> {
  try {
    await conversationService.sendMessage(jid, body);
  } catch (err) {
    console.error('[whatsapp.controller] fallback sendMessage failed:', err);
  }
}
