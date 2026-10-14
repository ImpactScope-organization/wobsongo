import type { NextFunction, Request, Response } from 'express';
import twilio from 'twilio';
import { env } from '../config/env.js';

// twilioSignature rejects webhook calls that Twilio did not sign.
export function twilioSignature(req: Request, res: Response, next: NextFunction): void {
  if (!env.twilioValidateSignature) {
    next();
    return;
  }

  const signature = req.header('X-Twilio-Signature') ?? '';
  const url = `${env.publicBaseUrl}${req.originalUrl}`;
  const valid = twilio.validateRequest(
    env.twilioAuthToken,
    signature,
    url,
    req.body as Record<string, string>
  );

  if (!valid) {
    console.warn('[twilio-signature] invalid signature, request rejected');
    res.sendStatus(403);
    return;
  }
  next();
}
