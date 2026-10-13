const MIME_TO_FORMAT: Record<string, string> = {
  'audio/ogg': 'ogg',
  'audio/opus': 'ogg',
  'audio/mpeg': 'mp3',
  'audio/mp3': 'mp3',
  'audio/mp4': 'm4a',
  'audio/aac': 'aac',
  'audio/amr': 'amr',
  'audio/webm': 'webm',
  'audio/wav': 'wav',
  'audio/x-wav': 'wav',
};

// formatFromMime maps a MIME type to the file
// extension the Modal ASR endpoint uses as a format hint.
export function formatFromMime(mime: string): string {
  const base = (mime.split(';')[0] ?? '').trim().toLowerCase();
  return MIME_TO_FORMAT[base] ?? 'ogg';
}

// formatFromPath does the same from a file path, Telegram voice notes use .oga.
export function formatFromPath(filePath: string): string {
  const ext = (filePath.split('.').pop() ?? '').toLowerCase();
  if (ext === '' || ext === 'oga' || ext === 'opus') return 'ogg';
  return ext;
}
