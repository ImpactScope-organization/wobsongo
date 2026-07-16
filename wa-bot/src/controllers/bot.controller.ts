import type { Request, Response } from 'express';
import * as botService from '../services/bot.service.js';
import type { StopBotRequest } from '../types/types.js';

// startBot handles the HTTP request to initialize and start the bot instance.
export async function startBot(_req: Request, res: Response): Promise<void> {
  const status = await botService.start();
  res.json(status);
}

//stopBot handles the HTTP request to gracefully shut down the bot instance.
// It accepts an optional 'purgeData' flag in the request body, 
// which determines  whether the bot's local session data should be deleted upon stopping.
export async function stopBot(req: Request, res: Response): Promise<void> {
  const body = req.body as StopBotRequest;
  const status = await botService.stop(body.purgeData ?? false);
  res.json(status);
}

// getBotStatus handles the HTTP request to retrieve the current operational state of the bot.
export function getBotStatus(_req: Request, res: Response): void {
  res.json(botService.getStatus());
}