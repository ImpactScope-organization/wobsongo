import type { Cookies } from '@sveltejs/kit';
import type { TokenPair } from '../services/auth.js';

export const ACCESS_COOKIE = 'wobsongo_access';
export const REFRESH_COOKIE = 'wobsongo_refresh';

// Both tokens are httpOnly cookies: the browser never calls the API, the
// server does, so the tokens never need to be readable from JavaScript.
// SvelteKit adds `secure` itself (except on http://localhost).
const OPTIONS = { path: '/', httpOnly: true, sameSite: 'lax' } as const;

function secondsUntil(at: Date, now: Date): number {
	return Math.max(0, Math.floor((at.getTime() - now.getTime()) / 1000));
}

export function setTokenCookies(cookies: Cookies, tokens: TokenPair, now: Date): void {
	cookies.set(ACCESS_COOKIE, tokens.accessToken, {
		...OPTIONS,
		maxAge: secondsUntil(tokens.accessExpiresAt, now)
	});
	cookies.set(REFRESH_COOKIE, tokens.refreshToken, {
		...OPTIONS,
		maxAge: secondsUntil(tokens.refreshExpiresAt, now)
	});
}

export function clearTokenCookies(cookies: Cookies): void {
	cookies.delete(ACCESS_COOKIE, OPTIONS);
	cookies.delete(REFRESH_COOKIE, OPTIONS);
}
