/**
 * Backend repositories, used when API_HOST is an http(s) URL. Placeholders
 * until the client is generated from the Go backend's swagger.json
 * (`../swagger/swagger.json`), which has no dashboard endpoints yet: every
 * method answers 501, which pages show as "… is not available yet."
 *
 * TODO(openapi): generate a typed client from swagger.json, call it here, and
 * map its DTOs onto the domain types (`#lib/domain/*`). Map 4xx statuses as
 * each repository interface documents them (401, 403, 404, 409).
 */
import type { Repos } from '../../services/index.js';
import { createApiAccountRepo } from './account.js';
import { createApiAuthRepo } from './auth.js';
import { createApiPulseRepo } from './pulse.js';

export interface ApiOptions {
	/** Backend origin, e.g. "https://api.wobsongo.org". */
	baseUrl: string;
	/** SvelteKit's `event.fetch`, so calls go through `handleFetch`. */
	fetch: typeof fetch;
}

export function createApiRepos(options: ApiOptions): Repos {
	return {
		auth: createApiAuthRepo(options),
		account: createApiAccountRepo(options),
		pulse: createApiPulseRepo(options)
	};
}
