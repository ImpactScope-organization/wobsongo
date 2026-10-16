import { isRedirect, type RequestEvent } from '@sveltejs/kit';
import { describe, expect, it, vi } from 'vitest';
import { OVERVIEW_NOW } from '#lib/fixtures/overview.js';
import { profile } from '#lib/fixtures/settings.js';
import { fixedClock } from '#lib/utils/clock.js';
import { ACCESS_COOKIE, REFRESH_COOKIE } from './auth/cookies.js';
import { loginUrl, resolveSession, safeRedirectTarget } from './auth/session.js';
import { parseApiHost, servicesFactory } from './backend.js';
import { createHandle, createHandleFetch, routeGroup } from './hooks.js';
import { ACCESS_TOKEN_TTL_MS, REFRESH_TOKEN_TTL_MS } from './repos/stub/tokens.js';
import { DEMO_ACCOUNT_ID } from './repos/stub/index.js';
import { createFakeCookies, createTestBackend, signInDemo } from './testing/fakes.js';

describe('parseApiHost', () => {
	it('uses stubs for stub:// hosts', () => {
		expect(parseApiHost('stub://local')).toEqual({ kind: 'stub' });
	});

	it('uses the backend for http(s) URLs, without a trailing slash', () => {
		expect(parseApiHost('https://api.wobsongo.org/')).toEqual({
			kind: 'api',
			baseUrl: 'https://api.wobsongo.org'
		});
		expect(parseApiHost('http://localhost:8080')).toEqual({
			kind: 'api',
			baseUrl: 'http://localhost:8080'
		});
	});

	it('refuses to start without a usable value', () => {
		expect(() => parseApiHost(undefined)).toThrow(/API_HOST is not set/);
		expect(() => parseApiHost('ftp://files')).toThrow(/must start with/);
		expect(() => parseApiHost('localhost:8080')).toThrow(/must start with/);
	});
});

describe('servicesFactory', () => {
	it('shares one stub store between requests', async () => {
		const servicesFor = servicesFactory({ kind: 'stub' }, fixedClock(OVERVIEW_NOW));
		expect(servicesFor(fetch)).toBe(servicesFor(fetch));
	});

	it('builds backend services per request', async () => {
		const servicesFor = servicesFactory(
			{ kind: 'api', baseUrl: 'https://api.example.org' },
			fixedClock(OVERVIEW_NOW)
		);
		expect(servicesFor(fetch)).not.toBe(servicesFor(fetch));
	});
});

describe('setTokenCookies', () => {
	it('sets httpOnly, lax, site-wide cookies that expire with their tokens', async () => {
		const backend = createTestBackend();
		const { cookies } = await signInDemo(backend);

		expect(cookies.options.get(ACCESS_COOKIE)).toMatchObject({
			path: '/',
			httpOnly: true,
			sameSite: 'lax',
			maxAge: ACCESS_TOKEN_TTL_MS / 1000
		});
		expect(cookies.options.get(REFRESH_COOKIE)).toMatchObject({
			httpOnly: true,
			maxAge: REFRESH_TOKEN_TTL_MS / 1000
		});
	});
});

describe('resolveSession', () => {
	it('reads the session from the access token', async () => {
		const backend = createTestBackend();
		const { cookies } = await signInDemo(backend);
		const session = await resolveSession(cookies, backend.services.auth, backend.clock);
		expect(session).toEqual({
			scope: { accountId: DEMO_ACCOUNT_ID },
			name: profile.name
		});
	});

	it('is null without cookies', async () => {
		const backend = createTestBackend();
		expect(
			await resolveSession(createFakeCookies(), backend.services.auth, backend.clock)
		).toBeNull();
	});

	it('refreshes an expired access token and stores the new one', async () => {
		const backend = createTestBackend();
		const { cookies } = await signInDemo(backend);
		const oldAccess = cookies.get(ACCESS_COOKIE);
		backend.clock.advance(ACCESS_TOKEN_TTL_MS + 1000);

		const session = await resolveSession(cookies, backend.services.auth, backend.clock);
		expect(session?.name).toBe(profile.name);
		expect(cookies.get(ACCESS_COOKIE)).toBeDefined();
		expect(cookies.get(ACCESS_COOKIE)).not.toBe(oldAccess);
	});

	it('clears the cookies when the refresh token is rejected', async () => {
		const backend = createTestBackend();
		const { cookies } = await signInDemo(backend);
		backend.clock.advance(REFRESH_TOKEN_TTL_MS);

		expect(await resolveSession(cookies, backend.services.auth, backend.clock)).toBeNull();
		expect(cookies.get(ACCESS_COOKIE)).toBeUndefined();
		expect(cookies.get(REFRESH_COOKIE)).toBeUndefined();
	});

	it('keeps the cookies when the backend is down', async () => {
		const backend = createTestBackend();
		const { cookies } = await signInDemo(backend);
		cookies.delete(ACCESS_COOKIE, { path: '/' });
		vi.spyOn(backend.services.auth, 'refresh').mockResolvedValueOnce({
			ok: false,
			status: 503,
			message: 'down'
		});

		expect(await resolveSession(cookies, backend.services.auth, backend.clock)).toBeNull();
		expect(cookies.get(REFRESH_COOKIE)).toBeDefined();
	});
});

describe('loginUrl', () => {
	it('comes back to the page after signing in', () => {
		expect(loginUrl(new URL('https://x.org/settings?tab=1'))).toBe(
			'/login?redirectTo=%2Fsettings%3Ftab%3D1'
		);
		expect(loginUrl(new URL('https://x.org/'))).toBe('/login');
	});
});

describe('safeRedirectTarget', () => {
	it('keeps paths on this site', () => {
		expect(safeRedirectTarget('/settings?tab=1#password')).toBe('/settings?tab=1#password');
	});

	it('falls back to home for anything else', () => {
		for (const target of [
			null,
			'',
			'settings',
			'https://evil.example/',
			'//evil.example/',
			'/\\evil.example/',
			'/\t/evil.example'
		]) {
			expect(safeRedirectTarget(target), String(target)).toBe('/');
		}
	});
});

describe('routeGroup', () => {
	it('tells app routes from sign-in routes', () => {
		expect(routeGroup('/(app)')).toBe('app');
		expect(routeGroup('/(app)/settings')).toBe('app');
		expect(routeGroup('/login')).toBe('auth');
		expect(routeGroup('/signup')).toBe('auth');
		expect(routeGroup('/logout')).toBeNull();
		expect(routeGroup(null)).toBeNull();
	});
});

describe('handle', () => {
	async function run(routeId: string | null, path: string, signedIn: boolean) {
		const backend = createTestBackend();
		const cookies = signedIn ? (await signInDemo(backend)).cookies : createFakeCookies();
		const handle = createHandle({ clock: backend.clock, servicesFor: () => backend.services });
		const event = {
			cookies,
			fetch,
			locals: {},
			route: { id: routeId },
			url: new URL(path, 'https://dashboard.example')
		} as unknown as RequestEvent;
		const resolve = vi.fn(async () => new Response('page'));

		try {
			await handle({ event, resolve });
			return { event, redirect: null, resolved: resolve.mock.calls.length > 0 };
		} catch (thrown) {
			if (!isRedirect(thrown)) throw thrown;
			return { event, redirect: thrown.location, resolved: false };
		}
	}

	it('sends signed-out visitors of app pages to login, then back', async () => {
		expect((await run('/(app)/settings', '/settings', false)).redirect).toBe(
			'/login?redirectTo=%2Fsettings'
		);
		expect((await run('/(app)', '/', false)).redirect).toBe('/login');
	});

	it('lets signed-in people through, with locals set', async () => {
		const { event, resolved } = await run('/(app)', '/', true);
		expect(resolved).toBe(true);
		expect(event.locals.session?.name).toBe(profile.name);
		expect(event.locals.services).toBeDefined();
		expect(event.locals.clock.now()).toEqual(OVERVIEW_NOW);
	});

	it('sends signed-in people away from login and signup', async () => {
		expect((await run('/login', '/login', true)).redirect).toBe('/');
		expect((await run('/signup', '/signup', true)).redirect).toBe('/');
	});

	it('leaves other routes alone', async () => {
		expect((await run('/login', '/login', false)).resolved).toBe(true);
		expect((await run(null, '/nope', false)).resolved).toBe(true);
	});
});

describe('handleFetch', () => {
	async function sent(baseUrl: string | null, url: string, init?: RequestInit) {
		const backend = createTestBackend();
		const { cookies } = await signInDemo(backend);
		const fetch = vi.fn(async () => new Response());
		const request = new Request(url, init);
		await createHandleFetch(baseUrl)({
			event: { cookies, getClientAddress: () => '203.0.113.7' } as unknown as RequestEvent,
			request,
			fetch: fetch as unknown as typeof globalThis.fetch
		});
		return request.headers;
	}
	const sentAuthorization = async (baseUrl: string | null, url: string) =>
		(await sent(baseUrl, url)).get('authorization');

	it('keeps an authorization the request sets itself', async () => {
		const headers = await sent('https://api.example.org', 'https://api.example.org/api/v1/me', {
			headers: { authorization: 'Bearer fresh' }
		});
		expect(headers.get('authorization')).toBe('Bearer fresh');
	});

	it("passes the visitor's IP to the backend only", async () => {
		const backend = await sent('https://api.example.org', 'https://api.example.org/api/v1/me');
		expect(backend.get('x-forwarded-for')).toBe('203.0.113.7');
		const other = await sent('https://api.example.org', 'https://other.example/x');
		expect(other.get('x-forwarded-for')).toBeNull();
	});

	it('sends the access token to the backend', async () => {
		expect(
			await sentAuthorization('https://api.example.org', 'https://api.example.org/api/v1/me')
		).toMatch(/^Bearer stub-access-/);
	});

	it('never sends it anywhere else', async () => {
		expect(
			await sentAuthorization('https://api.example.org', 'https://api.example.org.evil/x')
		).toBeNull();
		expect(
			await sentAuthorization('https://api.example.org', 'https://other.example/x')
		).toBeNull();
		expect(await sentAuthorization(null, 'https://api.example.org/x')).toBeNull();
	});
});
