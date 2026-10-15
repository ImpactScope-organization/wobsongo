import createClient from 'openapi-fetch';
import { failure, ok, type Result } from '../../result.js';
import type { TokenPair } from '../../services/auth.js';
import type { ApiOptions } from './index.js';
import type { components, paths } from './schema.js';

export type Schemas = components['schemas'];

/** A typed client for the Go backend, generated from its swagger.json (`pnpm gen:api`). */
export function createApiClient({ baseUrl, fetch }: ApiOptions) {
	return createClient<paths>({ baseUrl, fetch });
}

/** Every backend response is an envelope: `data` on success, `error` otherwise. */
interface Envelope<T> {
	data?: T;
	error?: string;
}

/**
 * Turns a client call into a Result, mapping the envelope's `data`. A failed
 * response keeps its status and the backend's message. An unreachable backend
 * is a 503, so callers treat it as an outage, never as a rejected token.
 */
export async function call<T, U>(
	request: Promise<{ data?: Envelope<T>; error?: Envelope<unknown>; response: Response }>,
	map: (data: T) => U
): Promise<Result<U>> {
	let result: Awaited<typeof request>;
	try {
		result = await request;
	} catch {
		return failure(503, 'The backend could not be reached.');
	}
	const { data, error, response } = result;
	if (!response.ok) return failure(response.status, error?.error || response.statusText);
	if (data?.data === undefined) return failure(502, 'The backend sent an unexpected response.');
	return ok(map(data.data));
}

export function toTokenPair(tokens: Schemas['dto.TokenPairResponse']): TokenPair {
	return {
		accessToken: tokens.access_token,
		accessExpiresAt: new Date(tokens.access_expires_at),
		refreshToken: tokens.refresh_token,
		refreshExpiresAt: new Date(tokens.refresh_expires_at)
	};
}
