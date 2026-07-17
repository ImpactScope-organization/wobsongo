import { Router } from 'express';
import * as botController from '../controllers/bot.controller.js';
import * as conversationController from '../controllers/conversation.controller.js';
import * as callbackController from '../controllers/callback.controller.js';
import { pskAuth } from '../middleware/psk-auth.middleware.js';
import { env } from '../config/env.js';
import { asyncHandler } from '../utils/async-handler.js';

export const controlRouter: Router = Router();

const goAuth = pskAuth(env.goExtractPsk);
// Bot lifecycle

// Start the WhatsApp bot connection.
controlRouter.post('/bot/start', goAuth, asyncHandler(botController.startBot));

// Stop the WhatsApp bot connection.
controlRouter.post('/bot/stop', goAuth, asyncHandler(botController.stopBot));

// Check the current bot connection status.
controlRouter.get('/bot/status', goAuth, botController.getBotStatus);

// Conversation

// Send a message to a user from the bot.
controlRouter.post(
  '/users/:jid/messages',
  goAuth,
  asyncHandler(conversationController.sendMessage)
);

// Delete a message previously sent by the bot.
controlRouter.delete(
  '/users/:jid/messages/:messageId',
  goAuth,
  asyncHandler(conversationController.deleteMessage)
);

export const callbackRouter: Router = Router();
callbackRouter.post(
  '/callback/extract-done',
  pskAuth(env.botCallbackPsk),
  asyncHandler(callbackController.extractDone)
);
