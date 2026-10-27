/**
 * Backend repositories, used when API_HOST is an http(s) URL. They call the Go
 * backend through a client typed from its swagger.json (`pnpm gen:api` after
 * the backend's spec changes) and map its DTOs onto the domain types
 * (`#lib/domain/*`). Endpoints the backend doesn't have yet answer 501, which
 * pages show as "… is not available yet."
 */
import type { Repos } from '../../services/index.js';
import { createApiAccountRepo } from './account.js';
import { createApiAuthRepo } from './auth.js';
import { createApiMonitorRepo } from './monitor.js';
import { createApiOverviewRepo } from './overview.js';

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
		overview: createApiOverviewRepo(options),
		monitor: createApiMonitorRepo(options)
	};
}
