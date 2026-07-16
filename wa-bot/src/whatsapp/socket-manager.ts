import type { WASocket } from 'baileys';
import type { BotConnectionStatus } from '../types/types.js';

// Holds the active Baileys WA socket connection.
let sock: WASocket | undefined;

// Tracks the current lifecycle/connection status of the bot.
let status: BotConnectionStatus = 'stopped';

// Stores the most recent QR code, if authentication is required.
let lastQr: string | undefined;

// Connecting or reconnecting to the WA servers.
export function setSock(newSock: WASocket | undefined): void {
  sock = newSock;
}

// getSock retrieves the active WA socket instance.
export function getSock(): WASocket | undefined {
  return sock;
}

// updates the current operational connection state of the bot.
// If the bot requires the user to scan a QR code for authentication, 
// the QR can be provided and stored in memory.
export function setStatus(newStatus: BotConnectionStatus, qr?: string): void {
  status = newStatus;
  lastQr = qr;
}

// getStatus retrieves the current connection state of the bot, 
export function getStatus(): { status: BotConnectionStatus; qr?: string } {
  return lastQr === undefined ? { status } : { status, qr: lastQr };
}