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
import { getSock, setSock, setStatus } from './socket-manager.js';
import { rm } from 'node:fs/promises';

const logger = P({ level: 'silent' });
const RECONNECT_DELAY_MS = 3000;

// OnMessage defines the callback signature used to process incoming WA messages.
type OnMessage = (sock: WASocket, msg: WAMessage) => Promise<void>;

// Wraps an async event listener so its rejection is caught and logged.
function safeListener<Args extends unknown[]>(
  fn: (...args: Args) => Promise<void>,
  label: string
): (...args: Args) => void {
  return (...args: Args) => {
    fn(...args).catch((err: unknown) => {
      console.error(`[socket] error in ${label} listener:`, err);
    });
  };
}

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
  sock.ev.on('creds.update', safeListener(saveCreds, 'creds.update'));

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
      const statusCode = (lastDisconnect?.error as Boom)?.output?.statusCode as
        DisconnectReason | undefined;
      const shouldReconnect = statusCode !== DisconnectReason.loggedOut;
      setStatus('disconnected');

      // Attempt to automatically reconnect unless the user explicitly logged out.
      if (shouldReconnect) {
        console.log(`Connection closed. Reconnecting in ${RECONNECT_DELAY_MS}ms...`);
        setTimeout(() => {
          startSocket(onMessage).catch((err) => {
            console.error('[socket] reconnect failed:', err);
            setStatus('disconnected');
          });
        }, RECONNECT_DELAY_MS);
      } else {
        console.log('Logged out. Delete the auth_info_baileys folder and run /bot/start again.');
        setSock(undefined);
        setStatus('stopped');
      }
    }
  });

  // Listen for new incoming messages
  sock.ev.on(
    'messages.upsert',
    safeListener(async ({ type, messages }) => {
      if (type !== 'notify') return;
      for (const msg of messages) await onMessage(sock, msg);
    }, 'messages.upsert')
  );
}

// stopSocket terminates the active WA socket connection.
export async function stopSocket(purgeData: boolean): Promise<void> {
  const sock = getSock();

  if (purgeData) {
    await sock?.logout().catch((err: unknown) => {
      console.error('[stopSocket] logout failed (continuing anyway):', err);
    });
  } else {
    await sock?.end(undefined)?.catch((err: unknown) => {
      console.error('[stopSocket] end failed (continuing anyway):', err);
    });
  }

  setSock(undefined);
  setStatus('stopped');

  // If requested, remove the session directory from the filesystem.
  if (purgeData) {
    await rm('auth_info_baileys', { recursive: true, force: true }).catch((err: unknown) => {
      console.error('[stopSocket] failed to purge session data:', err);
    });
  }
}
