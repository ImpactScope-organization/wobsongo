/**
 * Test doubles for calling hooks, loads and actions directly: an in-memory
 * cookie jar, and services on stub repositories with a controllable clock.
 */
import type { Cookies } from '@sveltejs/kit';
import { OVERVIEW_NOW } from '#lib/fixtures/overview.js';
import { manualClock, type ManualClock } from '#lib/testing/clock.js';
import { setTokenCookies } from '../auth/cookies.js';
import {
	createStubRepos,
	createStubStore,
	DEMO_EMAIL,
	DEMO_PASSWORD
} from '../repos/stub/index.js';
import type { StubStore } from '../repos/stub/store.js';
import { createServices, type Services, type Session } from '../services/index.js';

export interface FakeCookies extends Cookies {
	/** Options of the last `set` or `delete` per name, to check flags. */
	options: Map<string, Record<string, unknown>>;
}

export function createFakeCookies(initial: Record<string, string> = {}): FakeCookies {
	const jar = new Map(Object.entries(initial));
	const options = new Map<string, Record<string, unknown>>();
	return {
		options,
		get: (name) => jar.get(name),
		getAll: () => [...jar].map(([name, value]) => ({ name, value })),
		set: (name, value, opts = {}) => {
			// A cookie set to expire right away is gone, as in a browser.
			if (opts.maxAge === 0) jar.delete(name);
			else jar.set(name, value);
			options.set(name, { ...opts });
		},
		delete: (name, opts) => {
			jar.delete(name);
			options.set(name, { ...opts, deleted: true });
		},
		serialize: (name, value) => `${name}=${value}`,
		parse: () => {
			throw new Error('Not used in tests.');
		}
	};
}

export interface TestBackend {
	clock: ManualClock;
	store: StubStore;
	services: Services;
}

/** Fresh stub services, starting at the fixtures' "now". */
export function createTestBackend(start: Date = OVERVIEW_NOW): TestBackend {
	const clock = manualClock(start);
	const store = createStubStore();
	return { clock, store, services: createServices(createStubRepos(clock, store)) };
}

/** Signs the demo reviewer in: cookies holding their tokens, and their session. */
export async function signInDemo(
	backend: TestBackend
): Promise<{ cookies: FakeCookies; session: Session }> {
	const tokens = await backend.services.auth.login({ email: DEMO_EMAIL, password: DEMO_PASSWORD });
	if (!tokens.ok) throw new Error(tokens.message);
	const cookies = createFakeCookies();
	setTokenCookies(cookies, tokens.data, backend.clock.now());

	const session = await backend.services.auth.getSession(tokens.data.accessToken);
	if (!session.ok) throw new Error(session.message);
	return { cookies, session: session.data };
}

/** `locals` as the hook sets them. */
export function testLocals(backend: TestBackend, session: Session | null): App.Locals {
	return { clock: backend.clock, services: backend.services, session };
}

/** A posted form, as an action receives it. */
export function formRequest(url: string, fields: Record<string, string>): Request {
	const body = new FormData();
	for (const [name, value] of Object.entries(fields)) body.set(name, value);
	return new Request(url, { method: 'POST', body });
}
