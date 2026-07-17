import express, { ErrorRequestHandler } from 'express';
import { controlRouter, callbackRouter } from './routers/control.router.js';
import { env } from './config/env.js';

const errorHandler: ErrorRequestHandler = (err, _req, res, _next) => {
  console.error('[unhandled error]', err);
  res.status(500).json({ error: 'Internal server error' });
};

export function createServer() {
  const app = express();
  app.use(express.json());
  app.use(controlRouter);
  app.use(callbackRouter); // Mount the callback router to handle external webhook notifications\
  app.use(errorHandler);
  return app.listen(env.port, () => console.log(`Listening on port ${env.port}`));
}

// Initialize and start the bot server.
createServer();

process.on('unhandledRejection', (reason) => {
  console.error('[unhandledRejection]', reason);
});
process.on('uncaughtException', (err) => {
  console.error('[uncaughtException]', err);
});
