import { randomBytes } from 'node:crypto';

const TEMP_MEDIA_TTL_MS = 10 * 60 * 1000;

interface StoredMedia {
  buffer: Buffer;
  contentType: string;
  expiresAt: number;
}

// In-memory store for reply audio that Twilio must fetch by URL.
const media = new Map<string, StoredMedia>();

export function saveTempMedia(buffer: Buffer, contentType: string): string {
  const token = randomBytes(24).toString('hex');
  media.set(token, { buffer, contentType, expiresAt: Date.now() + TEMP_MEDIA_TTL_MS });
  return token;
}

export function getTempMedia(token: string): { buffer: Buffer; contentType: string } | undefined {
  const item = media.get(token);
  if (!item) return undefined;
  if (Date.now() > item.expiresAt) {
    media.delete(token);
    return undefined;
  }
  return { buffer: item.buffer, contentType: item.contentType };
}

setInterval(() => {
  const now = Date.now();
  for (const [token, item] of media) {
    if (now > item.expiresAt) media.delete(token);
  }
}, 60 * 1000).unref();
