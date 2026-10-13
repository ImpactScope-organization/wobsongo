import { env } from '../config/env.js';

export async function sendMessage(chatId: string, text: string): Promise<void> {
  if (!env.telegramBotToken) {
    throw new Error('[telegram] TELEGRAM_BOT_TOKEN is not set');
  }

  const url = `https://api.telegram.org/bot${env.telegramBotToken}/sendMessage`;

  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ chat_id: chatId, text }),
  });

  if (!response.ok) {
    const errBody = await response.text().catch(() => '');
    throw new Error(`[telegram] failed to send message (${response.status}): ${errBody}`);
  }

  console.log(`[telegram] sent message to ${chatId}`);
}

export async function sendVoice(chatId: string, audio: Buffer): Promise<void> {
  if (!env.telegramBotToken) {
    throw new Error('[telegram] TELEGRAM_BOT_TOKEN is not set');
  }

  const form = new FormData();
  form.append('chat_id', chatId);
  form.append('voice', new Blob([new Uint8Array(audio)], { type: 'audio/ogg' }), 'reply.ogg');

  const response = await fetch(`https://api.telegram.org/bot${env.telegramBotToken}/sendVoice`, {
    method: 'POST',
    body: form,
  });

  if (!response.ok) {
    const errBody = await response.text().catch(() => '');
    throw new Error(`[telegram] failed to send voice (${response.status}): ${errBody}`);
  }

  console.log(`[telegram] sent voice to ${chatId}`);
}
