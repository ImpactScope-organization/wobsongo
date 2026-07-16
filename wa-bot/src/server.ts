import express from 'express';
import { controlRouter, callbackRouter } from './routers/control.router.js';
import { env } from './config/env.js';

export function createServer() {
  const app = express();
  app.use(express.json());
  app.use(controlRouter);
  app.use(callbackRouter); // Mount the callback router to handle external webhook notifications
  return app.listen(env.port, () => console.log(`Listening on port ${env.port}`));
}

// Initialize and start the bot server.
createServer();