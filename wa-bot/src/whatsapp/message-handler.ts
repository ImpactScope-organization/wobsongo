import type { WASocket, WAMessage } from 'baileys';
import { callGoAgentInbound } from '../services/go-client.service.js';
import { savePendingJob } from '../services/pending-job.store.js';

export async function handleMessage(sock: WASocket, msg: WAMessage): Promise<void> {
  if (msg.key.fromMe || !msg.key.remoteJid) return;

  const jid = msg.key.remoteJid;
  const text = msg.message?.conversation ?? msg.message?.extendedTextMessage?.text;
  if (!text) return;

  try {
    await sock.sendPresenceUpdate('composing', jid);

    const result = await callGoAgentInbound({
      jid,
      text: text.trim()
    });

    if (result.status === 'rejected') {
      await sock.sendMessage(jid, { text: result.message ?? 'Sorry, I cannot process that.' });
      return;
    }

    // await sock.sendMessage(jid, { text: "⏳ Please wait, I'm checking this for you..." });
    savePendingJob(result.jobId, { jid, waitingMessageId: '', url: '' });
  } catch (err) {
    console.error('[message-handler] failed to call /agent/inbound:', err);
    await sock.sendMessage(jid, {
      text: '❌ An error occurred while contacting the server. Please try again later.',
    });
  }
}
