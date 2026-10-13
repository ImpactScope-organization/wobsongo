import express, { Router } from 'express';
import * as callbackController from '../controllers/callback.controller.js';
import * as ttsCallbackController from '../controllers/tts-callback.controller.js';
import { pskAuth } from '../middleware/psk-auth.middleware.js';
import { twilioSignature } from '../middleware/twilio-signature.middleware.js';
import { env } from '../config/env.js';
import { asyncHandler } from '../utils/async-handler.js';
import * as whatsappController from '../controllers/whatsapp.controller.js';
import * as telegramController from '../controllers/telegram.controller.js';
import * as mediaController from '../controllers/media.controller.js';

export const controlRouter: Router = Router();

export const callbackRouter: Router = Router();
callbackRouter.post(
  '/callback/extract-done',
  pskAuth(env.botCallbackPsk),
  asyncHandler(callbackController.extractDone)
);

// Result of a background speech job.
callbackRouter.post(
  '/callback/tts-done',
  pskAuth(env.botCallbackPsk),
  express.raw({ type: () => true, limit: '50mb' }),
  ttsCallbackController.ttsDone
);

// WhatsApp webhook endpoint for incoming messages
controlRouter.post(
  '/webhook',
  twilioSignature,
  asyncHandler(whatsappController.handleIncomingMessage)
);

// Telegram webhook endpoint for incoming messages
controlRouter.post('/telegram-webhook', asyncHandler(telegramController.handleTelegramWebhook));

// Short-lived reply audio that Twilio fetches by URL
controlRouter.get('/media/:file', mediaController.serveTempMedia);
