import { env } from '../config/env.js';

interface ModalAsrResponse {
  transcript?: string;
  language_detected?: string;
  error?: string;
}

// transcribeAudio sends the audio (base64) straight to the Modal ASR endpoint.
// Modal decodes it into a temp file, runs ffmpeg and the model.
// Returns the transcript, possibly empty if no speech was recognised.
export async function transcribeAudio(buffer: Buffer, format: string): Promise<string> {
  const res = await fetch(env.modalAsrEndpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      model: env.modalAsrModel,
      source_lang: env.modalAsrSourceLang,
      audio_base64: buffer.toString('base64'),
      audio_format: format,
    }),
    signal: AbortSignal.timeout(env.modalAsrTimeoutMs),
  });

  if (!res.ok) {
    throw new Error(`Modal ASR error: ${res.status} ${await res.text().catch(() => '')}`);
  }

  const data = (await res.json()) as ModalAsrResponse;
  if (data.error) throw new Error(`Modal ASR error: ${data.error}`);

  return (data.transcript ?? '').trim();
}
