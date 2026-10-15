import type { Clock } from '#lib/utils/clock.js';
import type { TokenPair } from '../../services/auth.js';
import type { StubStore } from './store.js';

/** Token lifetimes, as the backend issues them: 15 min access, 3-day refresh. */
export const ACCESS_TOKEN_TTL_MS = 15 * 60_000;
export const REFRESH_TOKEN_TTL_MS = 3 * 24 * 60 * 60_000;

/** A new access token tied to `refreshToken`, so logout revokes both. */
export function issueAccess(
	store: StubStore,
	clock: Clock,
	accountId: string,
	refreshToken: string
): Pick<TokenPair, 'accessToken' | 'accessExpiresAt'> {
	const accessToken = `stub-access-${crypto.randomUUID()}`;
	const accessExpiresAt = new Date(clock.now().getTime() + ACCESS_TOKEN_TTL_MS);
	store.accessTokens.set(accessToken, { accountId, expiresAt: accessExpiresAt, refreshToken });
	return { accessToken, accessExpiresAt };
}

export function issueTokens(store: StubStore, clock: Clock, accountId: string): TokenPair {
	const refreshToken = `stub-refresh-${crypto.randomUUID()}`;
	const refreshExpiresAt = new Date(clock.now().getTime() + REFRESH_TOKEN_TTL_MS);
	store.refreshTokens.set(refreshToken, { accountId, expiresAt: refreshExpiresAt });
	return { ...issueAccess(store, clock, accountId, refreshToken), refreshToken, refreshExpiresAt };
}

/** Revokes every token of the account, as a password change does on the backend. */
export function revokeAccountTokens(store: StubStore, accountId: string): void {
	for (const tokens of [store.accessTokens, store.refreshTokens]) {
		for (const [token, { accountId: owner }] of tokens) {
			if (owner === accountId) tokens.delete(token);
		}
	}
}
