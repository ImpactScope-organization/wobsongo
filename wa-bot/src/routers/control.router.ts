import { Router } from 'express';
import * as conversationController from '../controllers/conversation.controller.js';
import * as callbackController from '../controllers/callback.controller.js';
import { pskAuth } from '../middleware/psk-auth.middleware.js';
import { env } from '../config/env.js';
import { asyncHandler } from '../utils/async-handler.js';
import * as whatsappController from '../controllers/whatsapp.controller.js';

export const controlRouter: Router = Router();

const goAuth = pskAuth(env.goExtractPsk);

// Send a message to a user from the bot.
controlRouter.post(
  '/users/:jid/messages',
  goAuth,
  asyncHandler(conversationController.sendMessage)
);

export const callbackRouter: Router = Router();
callbackRouter.post(
  '/callback/extract-done',
  pskAuth(env.botCallbackPsk),
  asyncHandler(callbackController.extractDone)
);

controlRouter.post('/webhook', asyncHandler(whatsappController.handleIncomingMessage));
