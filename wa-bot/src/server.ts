import express, { ErrorRequestHandler } from 'express';
import { controlRouter, callbackRouter } from './routers/control.router.js';
import { env } from './config/env.js';
import * as Sentry from '@sentry/node';

if (env.sentryDsn) {
  Sentry.init({ dsn: env.sentryDsn });
}

const errorHandler: ErrorRequestHandler = (err, _req, res, next) => {
  console.error('[unhandled error]', err);
  if (res.headersSent) {
    next(err);
    return;
  }
  res.status(500).json({ error: 'Internal server error' });
};

export function createServer() {
  const app = express();
  app.use(express.json());
  app.use(express.urlencoded({ extended: true }));
  app.use(controlRouter);
  app.use(callbackRouter); // Mount the callback router to handle external webhook notifications
  Sentry.setupExpressErrorHandler(app);
  app.use(errorHandler);

  const server = app.listen(env.port, () => {
    console.log(`Listening on port ${env.port}`);
  });

  server.on('error', (err) => {
    console.error('[server] failed to start:', err);
    process.exit(1);
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
