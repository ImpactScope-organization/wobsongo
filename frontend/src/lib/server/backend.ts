/**
 * Picks the repositories behind the services from API_HOST: `stub://…` runs
 * on stub data, an http(s) URL on the backend.
 */
import type { Clock } from '#lib/utils/clock.js';
import { createApiRepos } from './repos/api/index.js';
import { createStubRepos } from './repos/stub/index.js';
import { createServices, type Services } from './services/index.js';

export type Backend = { kind: 'stub' } | { kind: 'api'; baseUrl: string };

/** Throws when API_HOST is missing or not a stub:// or http(s):// URL. */
export function parseApiHost(value: string | undefined): Backend {
	if (!value) {
		throw new Error(
			'API_HOST is not set. Use stub://local for stub data, or the backend URL, e.g. https://api.wobsongo.org.'
		);
	}
	if (value.startsWith('stub://')) return { kind: 'stub' };

	const url = URL.parse(value);
	if (!url || (url.protocol !== 'http:' && url.protocol !== 'https:')) {
		throw new Error(`API_HOST must start with stub://, http:// or https://, got "${value}".`);
	}
	return { kind: 'api', baseUrl: url.href.replace(/\/$/, '') };
}

/**
 * Services for one request. Stub services are shared by every request, so
 * their in-memory store outlives it; backend services use the request's fetch.
 */
export function servicesFactory(
	backend: Backend,
	clock: Clock
): (fetch: typeof globalThis.fetch) => Services {
	if (backend.kind === 'stub') {
		const services = createServices(createStubRepos(clock));
		return () => services;
	}
	return (fetch) => createServices(createApiRepos({ baseUrl: backend.baseUrl, fetch }));
}
