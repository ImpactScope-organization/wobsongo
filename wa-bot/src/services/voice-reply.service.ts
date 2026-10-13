import { env } from '../config/env.js';
import { callGoTTS } from './go-client.service.js';
import * as conversationService from './conversation.service.js';
import * as telegramService from './telegram.service.js';
import { saveTempMedia } from './temp-media.store.js';

// requestVoiceReply asks the backend to synthesize the answer in the background.
// Returns true if the backend accepted the job.
export async function requestVoiceReply(
  jobId: string,
  platform: 'whatsapp' | 'telegram',
  text: string
): Promise<boolean> {
  const language =
    platform === 'telegram' ? env.voiceReplyLangTelegram : env.voiceReplyLangWhatsapp;

  try {
    await callGoTTS({ jobId, text, language });
    return true;
  } catch (err) {
    console.error(`[voice-reply] backend rejected the TTS request (${platform}):`, err);
    return false;
  }
}

// deliverVoiceReply sends finished OGG/Opus audio to the user.
export async function deliverVoiceReply(
  platform: 'whatsapp' | 'telegram',
  jid: string,
  audio: Buffer
): Promise<void> {
  if (platform === 'telegram') {
    await telegramService.sendVoice(jid, audio);
    return;
  }

  const token = saveTempMedia(audio, 'audio/ogg');
  await conversationService.sendMedia(jid, `${env.publicBaseUrl}/media/${token}.ogg`);
}
