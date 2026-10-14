import type { Request, Response } from 'express';
import { handleTtsDone } from '../services/extract-callback.service.js';

interface TtsDoneBody {
  jobId?: string;
  status?: string;
  audioUrl?: string;
}

function isHttpsUrl(value: string | undefined): value is string {
  if (!value) return false;
  try {
    return new URL(value).protocol === 'https:';
  } catch {
    return false;
  }
}

// ttsDone handles the backend's callback with the result of a speech job. The
// audio stays in S3: a completed job carries a presigned URL the platform
// (Twilio/Telegram) fetches itself.
export function ttsDone(req: Request, res: Response): void {
  const { jobId, status, audioUrl } = (req.body ?? {}) as TtsDoneBody;

  if (!jobId || (status !== 'completed' && status !== 'failed')) {
    res.status(400).json({ error: 'jobId and a valid status are required' });
    return;
  }
  if (status === 'completed' && !isHttpsUrl(audioUrl)) {
    res.status(400).json({ error: 'a completed job needs an https audioUrl' });
    return;
  }

  // Answer right away so the backend does not wait on delivery to the user.
  res.sendStatus(204);

  handleTtsDone(jobId, status, audioUrl ?? '').catch((err) => {
    console.error('[tts-callback.controller] background processing error:', err);
  });
}
