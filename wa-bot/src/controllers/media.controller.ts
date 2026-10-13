import type { Request, Response } from 'express';
import { getTempMedia } from '../services/temp-media.store.js';

// serveTempMedia serves reply audio to Twilio.
export function serveTempMedia(req: Request, res: Response): void {
  const raw = req.params.file;
  const file = Array.isArray(raw) ? (raw[0] ?? '') : (raw ?? '');
  const token = file.replace(/\.ogg$/, '');

  const item = getTempMedia(token);
  if (!item) {
    res.sendStatus(404);
    return;
  }
  res.set('Cache-Control', 'no-store');
  res.type(item.contentType).send(item.buffer);
}
