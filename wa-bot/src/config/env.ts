import 'dotenv/config';
import type { ReplyLang } from '../types/types.js';

function required(name: string): string {
  const value = process.env[name];
  if (!value) throw new Error(`Env "${name}" not set`);
  return value;
}

function replyLang(name: string, fallback: ReplyLang): ReplyLang {
  const value = process.env[name]?.toLowerCase();
  return value === 'fr' || value === 'en' ? value : fallback;
}

export const env = {
  port: Number(process.env.PORT) || 3000,

  // The Pre-Shared Key used to authenticate outgoing requests to the Go backend.
  goExtractPsk: required('GO_EXTRACT_PSK'),

  // The Pre-Shared Key used to validate incoming callback requests from the Go backend.
  botCallbackPsk: required('BOT_CALLBACK_PSK'),

  // The base URL of the Go backend service
  goBackendUrl: required('GO_BACKEND_URL'),

  // Twilio config
  twilioAccountSid: required('TWILIO_ACCOUNT_SID'),
  twilioAuthToken: required('TWILIO_AUTH_TOKEN'),
  twilioPhoneNumber: required('TWILIO_PHONE_NUMBER'),

  // Validate X-Twilio-Signature on /webhook. Off by default.
  twilioValidateSignature: process.env.TWILIO_VALIDATE_SIGNATURE === 'true',

  // Telegram bot token
  telegramBotToken: process.env.TELEGRAM_BOT_TOKEN,

  // Public base URL of this bot. Twilio fetches reply audio from it.
  publicBaseUrl: required('PUBLIC_BASE_URL').replace(/\/+$/, ''),

  // Voice reply (TTS goes through the Go backend)
  voiceReplyWaitMs: Number(process.env.VOICE_REPLY_WAIT_MS) || 100 * 1000,
  voiceReplyLangWhatsapp: replyLang('VOICE_REPLY_LANG_WHATSAPP', 'fr'),
  voiceReplyLangTelegram: replyLang('VOICE_REPLY_LANG_TELEGRAM', 'en'),

  // Sentry DSN for error tracking
  sentryDsn: process.env.SENTRY_DSN,
};
