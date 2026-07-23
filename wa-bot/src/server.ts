import express from 'express';
import { controlRouter, callbackRouter } from './routers/control.router.js';
import { env } from './config/env.js';
import * as botService from './services/bot.service.js';

export function createServer() {
  const app = express();
  app.use(express.json());
  app.use(controlRouter);
  app.use(callbackRouter); // Mount the callback router to handle external webhook notifications\

  const server = app.listen(env.port, () => {
    console.log(`Listening on port ${env.port}`);
  });

  botService.start().catch((err) => {
    console.error('[server] WhatsApp connection auto-start failed:', err);
  });

  return server;
}

createServer();

process.on('unhandledRejection', (reason) => {
  console.error('[unhandledRejection]', reason);
});
process.on('uncaughtException', (err) => {
  console.error('[uncaughtException]', err);
});
