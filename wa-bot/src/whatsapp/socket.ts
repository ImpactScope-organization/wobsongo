import makeWASocket, {
  useMultiFileAuthState,
  fetchLatestBaileysVersion,
  DisconnectReason,
  type WASocket,
  type WAMessage,
} from 'baileys';
import { Boom } from '@hapi/boom';
import P from 'pino';
import qrcodeTerminal from 'qrcode-terminal';
import { setSock, setStatus } from './socket-manager.js';

const logger = P({ level: 'silent' });

// OnMessage defines the callback signature used to process incoming WA messages.
type OnMessage = (sock: WASocket, msg: WAMessage) => Promise<void>;

// startSocket initializes and manages the Baileys WA socket connection.
export async function startSocket(onMessage: OnMessage): Promise<void> {
  // Load existing credentials or initialize a new auth state directory.
  const { state, saveCreds } = await useMultiFileAuthState('auth_info_baileys');
  const { version } = await fetchLatestBaileysVersion();

  // Create the WA socket connection.
  const sock = makeWASocket({ version, auth: state, logger });
  setSock(sock);
  setStatus('starting');

  // Listen for credential updates and save them locally.
  sock.ev.on('creds.update', saveCreds);

  // Listen for changes in the connection lifecycle.
  sock.ev.on('connection.update', (update) => {
    const { connection, lastDisconnect, qr } = update;

    // If a QR code is generated, the bot needs to be linked to a WhatsApp account.
    if (qr) {
      console.log('Scan this QR code using WhatsApp (Linked Devices):');
      qrcodeTerminal.generate(qr, { small: true });
      setStatus('starting', qr);
    }

    // The connection has been successfully established.
    if (connection === 'open') {
      console.log(' WhatsApp connected.');
      setStatus('connected');
    }

    // The connection was closed or dropped.
    if (connection === 'close') {
      const statusCode = (lastDisconnect?.error as Boom)?.output?.statusCode;
      const shouldReconnect = statusCode !== DisconnectReason.loggedOut;
      setStatus('disconnected');

      // Attempt to automatically reconnect unless the user explicitly logged out.
      if (shouldReconnect) {
        startSocket(onMessage);
      } else {
        console.log('Logged out. Delete the auth_info_baileys folder and run /bot/start again.');
        setSock(undefined);
        setStatus('stopped');
      }
    }
  });

  // Listen for new incoming messages
  sock.ev.on('messages.upsert', async ({ type, messages }) => {
    if (type !== 'notify') return;
    for (const msg of messages) await onMessage(sock, msg);
  });
}

// stopSocket terminates the active WA socket connection.
export async function stopSocket(purgeData: boolean): Promise<void> {
  const { getSock } = await import('./socket-manager.js');
  const sock = getSock();

  // Attempt to safely log out of the active session.
  await sock?.logout().catch(() => {});
  setSock(undefined);
  setStatus('stopped');

  // If requested, remove the session directory from the filesystem.
  if (purgeData) {
    const { rm } = await import('node:fs/promises');
    await rm('auth_info_baileys', { recursive: true, force: true }).catch(() => {});
  }
}