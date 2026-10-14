import twilio from 'twilio';
import { env } from '../config/env.js';
import type { SendMessageRequest, SendMessageResponse } from '../types/types.js';

// initialize Twilio Client
const client = twilio(env.twilioAccountSid, env.twilioAuthToken);

export async function sendMessage(
  jid: string,
  body: SendMessageRequest
): Promise<SendMessageResponse> {
  const toPhone = `whatsapp:+${jid.replace('@s.whatsapp.net', '')}`;
  const message = await client.messages.create({
    from: `whatsapp:${env.twilioPhoneNumber}`,
    to: toPhone,
    body: body.text ?? '',
  });
  return { messageId: message.sid, sentAt: message.dateCreated.toISOString() };
}

// sendMedia sends a media-only WhatsApp message. Twilio fetches mediaUrl itself,
// so the URL must be publicly reachable.
export async function sendMedia(jid: string, mediaUrl: string): Promise<SendMessageResponse> {
  const toPhone = `whatsapp:+${jid.replace('@s.whatsapp.net', '')}`;
  const message = await client.messages.create({
    from: `whatsapp:${env.twilioPhoneNumber}`,
    to: toPhone,
    mediaUrl: [mediaUrl],
  });
  return { messageId: message.sid, sentAt: message.dateCreated.toISOString() };
}
