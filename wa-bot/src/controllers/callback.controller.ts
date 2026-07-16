import type { Request, Response } from 'express';
import { handleExtractDone } from '../services/extract-callback.service.js';
import type { ExtractDoneCallback } from '../types/types.js';

// extractDone handles the webhook callback from the backend indicating
// that a media extraction job has finished (either successfully or with an error).
export function extractDone(req: Request, res: Response): void {
    console.log('[extractDone] req.body:', JSON.stringify(req.body));
  const body = req.body as ExtractDoneCallback;

  if (!body.jobId || !body.status) {
    res.status(400).json({ error: 'obId and status are required' });
    return;
  }

  // send a response to the Go backend to prevent it from timing out.
  res.sendStatus(204);

  handleExtractDone(body.jobId, body.status, body.error).catch((err) => {
    console.error('[callback.controller] background processing error:', err);
  });
}