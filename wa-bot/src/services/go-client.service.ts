import { env } from '../config/env.js';
import type { ExtractResponse, ReplyLang } from '../types/types.js';

// APIEnvelope mirrors the standard response format returned by the Go backend.
interface APIEnvelope<T> {
  request_id: string;
  status: number;
  data: T;
  error?: string;
}

// callGoExtract triggers the POST /api/extract endpoint on the Go backend.
// Used twice within a single full flow, once when the initial link comes in,
// and again after receiving the callback with the same request.
export async function callGoExtract(payload: {
  url?: string;
  question?: string;
}): Promise<ExtractResponse> {
  const res = await fetch(`${env.goBackendUrl}/api/extract`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `PSK ${env.goExtractPsk}`,
    },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    throw new Error(`Go extract API error: ${res.status} ${await res.text().catch(() => '')}`);
  }

  const envelope = (await res.json()) as APIEnvelope<ExtractResponse>;
  return envelope.data;
}

export async function callGoAgentInbound(payload: {
  jid: string;
  text: string;
}): Promise<ExtractResponse> {
  const res = await fetch(`${env.goBackendUrl}/api/agent/inbound`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `PSK ${env.goExtractPsk}`,
    },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    throw new Error(
      `Go agent inbound API error: ${res.status} ${await res.text().catch(() => '')}`
    );
  }

  const envelope = (await res.json()) as APIEnvelope<ExtractResponse>;
  return envelope.data;
}

// callGoTTS asks the Go backend to start synthesizing a reply. It returns as
// soon as the backend accepts the job (202). The audio arrives later through
// the /callback/tts-done endpoint.
export async function callGoTTS(payload: {
  jobId: string;
  text: string;
  language: ReplyLang;
}): Promise<void> {
  const res = await fetch(`${env.goBackendUrl}/api/tts`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `PSK ${env.goExtractPsk}`,
    },
    body: JSON.stringify(payload),
    signal: AbortSignal.timeout(15_000),
  });

  if (!res.ok) {
    throw new Error(`Go TTS API error: ${res.status} ${await res.text().catch(() => '')}`);
  }
}

// callGoAgentInboundVoice forwards a reference to a voice message to the Go
// backend. The backend downloads the media itself, transcribes it and runs the
// agent turn, so the audio never passes through this service.
export async function callGoAgentInboundVoice(payload: {
  jid: string;
  channel: 'whatsapp' | 'telegram';
  mediaRef: string;
  mimeType: string;
}): Promise<ExtractResponse> {
  const res = await fetch(`${env.goBackendUrl}/api/agent/inbound-voice`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `PSK ${env.goExtractPsk}`,
    },
    body: JSON.stringify(payload),
    signal: AbortSignal.timeout(15_000),
  });

  if (!res.ok) {
    throw new Error(
      `Go agent inbound voice API error: ${res.status} ${await res.text().catch(() => '')}`
    );
  }

  const envelope = (await res.json()) as APIEnvelope<ExtractResponse>;
  return envelope.data;
}
