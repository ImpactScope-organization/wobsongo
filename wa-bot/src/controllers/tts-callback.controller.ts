import type { Request, Response } from 'express';
import { handleTtsDone } from '../services/extract-callback.service.js';

// ttsDone handles the backend's callback with the result of a speech job. The
// result is in headers and the body is the raw OGG/Opus audio (empty on failure).
export function ttsDone(req: Request, res: Response): void {
  const jobId = req.header('X-Job-Id');
  const status = req.header('X-Tts-Status');

  if (!jobId || (status !== 'completed' && status !== 'failed')) {
    res.status(400).json({ error: 'X-Job-Id and a valid X-Tts-Status are required' });
    return;
  }

  const audio: Buffer = Buffer.isBuffer(req.body) ? req.body : Buffer.alloc(0);

  // Answer right away so the backend does not wait on delivery to the user.
  res.sendStatus(204);

  handleTtsDone(jobId, status, audio).catch((err) => {
    console.error('[tts-callback.controller] background processing error:', err);
  });
}
