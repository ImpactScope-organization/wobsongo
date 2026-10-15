import { redirect, type Cookies } from '@sveltejs/kit';
import type { Clock } from '#lib/utils/clock.js';
import type { AuthService, Session } from '../services/auth.js';
import { ACCESS_COOKIE, REFRESH_COOKIE, clearTokenCookies, setTokenCookies } from './cookies.js';

/**
 * Who is signed in, from the token cookies. An expired access token (the
 * cookie expires with it) is replaced using the refresh token. Cookies are
 * cleared only when the backend rejects them, not when it is down.
 */
export async function resolveSession(
	cookies: Cookies,
	auth: AuthService,
	clock: Clock
): Promise<Session | null> {
	const accessToken = cookies.get(ACCESS_COOKIE);
	if (accessToken) {
		const session = await auth.getSession(accessToken);
		if (session.ok) return session.data;
	}

	const refreshToken = cookies.get(REFRESH_COOKIE);
	if (!refreshToken) return null;

	const tokens = await auth.refresh(refreshToken);
	if (!tokens.ok) {
		if (tokens.status === 401) clearTokenCookies(cookies);
		return null;
	}
	setTokenCookies(cookies, tokens.data, clock.now());

	const session = await auth.getSession(tokens.data.accessToken);
	return session.ok ? session.data : null;
}

/** Where to send someone who isn't signed in, coming back here afterwards. */
export function loginUrl(url: URL): string {
	const here = url.pathname + url.search;
	return here === '/' ? '/login' : `/login?redirectTo=${encodeURIComponent(here)}`;
}

const ORIGIN = 'http://internal.invalid';

/**
 * A `redirectTo` that stays on this site: a path, never another origin
 * (`//evil.example`, `/\evil.example`, `https://…`). Anything else is "/".
 */
export function safeRedirectTarget(value: string | null | undefined): string {
	if (!value?.startsWith('/')) return '/';
	const url = URL.parse(value, ORIGIN);
	if (!url || url.origin !== ORIGIN) return '/';
	return url.pathname + url.search + url.hash;
}

/**
 * The session for an `(app)` load or action. The hook already redirects
 * signed-out requests; this guards against it being bypassed.
 */
export function requireSession(locals: App.Locals, url: URL): Session {
	if (!locals.session) redirect(303, loginUrl(url));
	return locals.session;
}
