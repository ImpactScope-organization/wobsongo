import { startSocket, stopSocket } from '../whatsapp/socket.js';
import { handleMessage } from '../whatsapp/message-handler.js';
import { getStatus as getSocketStatus } from '../whatsapp/socket-manager.js';
import type { BotStatus } from '../types/types.js';

// start initializes and connects the WA socket if it is not already connected.
export async function start(): Promise<BotStatus> {
  if (getSocketStatus().status === 'connected') {
    return getSocketStatus();
  }

  // Start the socket connection and attach the message handler
  await startSocket(handleMessage);
  return getSocketStatus();
}

// stop disconnects the WA socket and halts bot operations.
// It can optionally delete the local session data, which is necessary if want
// to force a re-authentication (scan QR code again) on the next start.
export async function stop(purgeData: boolean): Promise<BotStatus> {
  await stopSocket(purgeData);
  return getSocketStatus();
}
// getStatus synchronously retrieves the current connection state of the WA bot.
export function getStatus(): BotStatus {
  return getSocketStatus();
}
