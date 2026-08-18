import 'dotenv/config';

function required(name: string): string {
  const value = process.env[name];
  if (!value) throw new Error(`Env "${name}" belum di-set`);
  return value;
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

  // Telegram bot token
  telegramBotToken: process.env.TELEGRAM_BOT_TOKEN,

  // Sentry DSN for error tracking
  sentryDsn: process.env.SENTRY_DSN,
};
