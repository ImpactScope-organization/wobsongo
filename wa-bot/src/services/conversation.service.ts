import { getSock } from '../whatsapp/socket-manager.js';
import type { SendMessageRequest, SendMessageResponse } from '../types/types.js';

// sendMessage dispatches a text message to a specific WA user.
export async function sendMessage(
  jid: string,
  body: SendMessageRequest
): Promise<SendMessageResponse> {
  const sock = getSock();
  if (!sock) throw new Error('The bot is not connected yet. Please run /bot/start first.');

  const remoteJid = jid.includes('@') ? jid : `${jid}@s.whatsapp.net`;

  // If a replyToMessageId is provided, configure the payload to quote that specific message
  const options = body.replyToMessageId
    ? {
        quoted: {
          key: { id: body.replyToMessageId, remoteJid, fromMe: true },
          message: {},
        },
      }
    : undefined;

  // Dispatch the message via the WA socket
  const sent = await sock.sendMessage(remoteJid, { text: body.text ?? '' }, options);

  return {
    messageId: sent?.key.id ?? '',
    sentAt: new Date().toISOString(),
  };
}

// deleteMessage deletes a previously sent message in a chat.
export async function deleteMessage(jid: string, messageId: string): Promise<void> {
  const sock = getSock();
  if (!sock) throw new Error('The bot is not connected yet.');

  const remoteJid = jid.includes('@') ? jid : `${jid}@s.whatsapp.net`;

  // Send the delete payload to the WA server
  await sock.sendMessage(remoteJid, {
    delete: { id: messageId, remoteJid, fromMe: true },
  });
}