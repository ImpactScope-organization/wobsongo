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
