import type { Request, Response } from 'express';
import * as conversationService from '../services/conversation.service.js';
import type { SendMessageRequest } from '../types/types.js';

function getParam(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

// sendMessage handles the HTTP request to send a message
export async function sendMessage(req: Request, res: Response): Promise<void> {
  const jid = getParam(req.params.jid);
  if (!jid) {
    res.status(400).json({ error: 'jid must be populated in path' });
    return;
  }

  const body = req.body as SendMessageRequest;
  if (!body.text) {
    res.status(400).json({ error: 'Tiktok URL is required.' });
    return;
  }

  const result = await conversationService.sendMessage(jid, body);
  res.json(result);
}

// deleteMessage handles the HTTP request to delete a specific message.
export async function deleteMessage(req: Request, res: Response): Promise<void> {
  const jid = getParam(req.params.jid);
  const messageId = getParam(req.params.messageId);
  if (!jid || !messageId) {
    res.status(400).json({ error: 'jid and messageId are required in the path.' });
    return;
  }

  await conversationService.deleteMessage(jid, messageId);
  res.sendStatus(204);
}
