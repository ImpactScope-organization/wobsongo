import { env } from '../config/env.js';
import { formatFromMime, formatFromPath } from '../utils/audio-format.js';

const MAX_AUDIO_BYTES = 16 * 1024 * 1024;
const DOWNLOAD_TIMEOUT_MS = 30_000;

export interface DownloadedAudio {
  buffer: Buffer;
  format: string;
}

async function readAudioBody(res: Response): Promise<Buffer> {
  const declared = Number(res.headers.get('content-length') ?? 0);
  if (declared > MAX_AUDIO_BYTES) throw new Error(`audio too large: ${declared} bytes`);

  const buffer = Buffer.from(await res.arrayBuffer());
  if (buffer.length > MAX_AUDIO_BYTES) throw new Error(`audio too large: ${buffer.length} bytes`);
  if (buffer.length === 0) throw new Error('downloaded audio is empty');
  return buffer;
}

function assertTwilioMediaUrl(rawUrl: string): URL {
  const url = new URL(rawUrl);
  const host = url.hostname.toLowerCase();
  if (url.protocol !== 'https:' || !(host === 'api.twilio.com' || host.endsWith('.twilio.com'))) {
    throw new Error(`refusing to download media from untrusted host: ${host}`);
  }
  return url;
}

// downloadTwilioMedia fetches an inbound WhatsApp voice note from Twilio.
export async function downloadTwilioMedia(
  mediaUrl: string,
  contentType: string
): Promise<DownloadedAudio> {
  const url = assertTwilioMediaUrl(mediaUrl);
  const basic = Buffer.from(`${env.twilioAccountSid}:${env.twilioAuthToken}`).toString('base64');

  const res = await fetch(url, {
    headers: { Authorization: `Basic ${basic}` },
    signal: AbortSignal.timeout(DOWNLOAD_TIMEOUT_MS),
  });
  if (!res.ok) throw new Error(`twilio media download failed: ${res.status}`);

  return { buffer: await readAudioBody(res), format: formatFromMime(contentType) };
}

interface TelegramGetFileResponse {
  ok: boolean;
  result?: { file_path?: string; file_size?: number };
}

// downloadTelegramFile fetches a Telegram voice note by file_id.
export async function downloadTelegramFile(fileId: string): Promise<DownloadedAudio> {
  const token = env.telegramBotToken;
  if (!token) throw new Error('[telegram] TELEGRAM_BOT_TOKEN is not set');

  const infoRes = await fetch(
    `https://api.telegram.org/bot${token}/getFile?file_id=${encodeURIComponent(fileId)}`,
    { signal: AbortSignal.timeout(DOWNLOAD_TIMEOUT_MS) }
  );
  if (!infoRes.ok) throw new Error(`telegram getFile failed: ${infoRes.status}`);

  const info = (await infoRes.json()) as TelegramGetFileResponse;
  const filePath = info.result?.file_path;
  if (!info.ok || !filePath) throw new Error('telegram getFile returned no file_path');

  const res = await fetch(`https://api.telegram.org/file/bot${token}/${filePath}`, {
    signal: AbortSignal.timeout(DOWNLOAD_TIMEOUT_MS),
  });
  if (!res.ok) throw new Error(`telegram file download failed: ${res.status}`);

  return { buffer: await readAudioBody(res), format: formatFromPath(filePath) };
}
