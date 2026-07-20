import { env } from '../config/env.js';
import type { ExtractResponse } from '../types/types.js';

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
export async function callGoExtract(url: string): Promise<ExtractResponse> {
  const res = await fetch(`${env.goBackendUrl}/api/extract`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `PSK ${env.goExtractPsk}`,
    },
    body: JSON.stringify({ url }),
  });

  if (!res.ok) {
    throw new Error(`Go extract API error: ${res.status} ${await res.text().catch(() => '')}`);
  }

  const envelope = (await res.json()) as APIEnvelope<ExtractResponse>;
  return envelope.data;
}