import type { WASocket, WAMessage } from 'baileys';
import { callGoExtract } from '../services/go-client.service.js';
import { savePendingJob } from '../services/pending-job.store.js';
import * as conversationService from '../services/conversation.service.js';

const TIKTOK_URL_REGEX = /https?:\/\/(www\.)?(vt\.)?tiktok\.com\/\S+/i;

// WELCOME_TEXT contains the default onboarding and help instructions
// when user send "/start" command.
const WELCOME_TEXT = `👋 Hello! I am a TikTok transcription bot.
How to use:
1. Send a TikTok video link to this chat
2. Wait for the bot to process it
3. The bot will reply with the transcription result
Type /start anytime to see this message again.`;

export async function handleMessage(sock: WASocket, msg: WAMessage): Promise<void> {
  // Ignore messages sent by the bot itself (fromMe) or messages without a valid sender JID.
  if (msg.key.fromMe || !msg.key.remoteJid) return;

  const jid = msg.key.remoteJid;
  const text = msg.message?.conversation ?? msg.message?.extendedTextMessage?.text;
  if (!text) return;

  try {
    const trimmed = text.trim();

    // Handle the basic /start command by sending the help text.
    if (trimmed === '/start') {
      await sock.sendMessage(jid, { text: WELCOME_TEXT });
      return;
    }

    // Attempt to extract a valid TikTok URL from the user's message.
    const tiktokUrl = trimmed.match(TIKTOK_URL_REGEX)?.[0];
    if (!tiktokUrl) {
      await sock.sendMessage(jid, {
        text: 'Please send a valid TikTok video link, or type /start for help..',
      });
      return;
    }

    // Initiate the extraction process by calling the Go backend.
    const result = await callGoExtract(tiktokUrl);

    if (result.status === 'completed' && result.data) {
      // If the data is already available. Reply immediately without sending a "wait" message.
      await conversationService.sendMessage(jid, {
        text: `📝 Transcription result:\n\n${result.data.transcript}`,
      });
      return;
    }

    if (result.status === 'failed') {
      await conversationService.sendMessage(jid, {
        text: `❌ Failed to process the video. ${result.error ?? ''}`.trim(),
      });
      return;
    }

    // If the status is 'processing' send a waiting message and store the job context.
    const waitingMsg = await conversationService.sendMessage(jid, {
      text: '⏳ Processing, please wait...',
    });

    savePendingJob(result.jobId, {
      jid,
      waitingMessageId: waitingMsg.messageId,
      url: tiktokUrl,
    });
  } catch (err) {
    console.error('[message-handler] failed to call /extract:', err);
    await sock.sendMessage(jid, {
      text: '❌ An error occurred while contacting the server. Please try again later.',
    });
  }
}
