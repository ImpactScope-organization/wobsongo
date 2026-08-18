import { Router } from 'express';
import * as callbackController from '../controllers/callback.controller.js';
import { pskAuth } from '../middleware/psk-auth.middleware.js';
import { env } from '../config/env.js';
import { asyncHandler } from '../utils/async-handler.js';
import * as whatsappController from '../controllers/whatsapp.controller.js';
import * as telegramController from '../controllers/telegram.controller.js';

export const controlRouter: Router = Router();

export const callbackRouter: Router = Router();
callbackRouter.post(
  '/callback/extract-done',
  pskAuth(env.botCallbackPsk),
  asyncHandler(callbackController.extractDone)
);

// WhatsApp webhook endpoint for incoming messages
controlRouter.post('/webhook', asyncHandler(whatsappController.handleIncomingMessage));

// Telegram webhook endpoint for incoming messages
controlRouter.post('/telegram-webhook', asyncHandler(telegramController.handleTelegramWebhook));
