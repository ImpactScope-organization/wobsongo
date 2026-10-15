import type { Clock } from '#lib/utils/clock.js';
import { failure, ok } from '../../result.js';
import type { AuthRepo, TokenPair } from '../../services/auth.js';
import { findAccountByEmail, simulatedFailure, type StubAccount, type StubStore } from './store.js';

/** Token lifetimes from tr.md: short-lived access, week-long refresh. */
export const ACCESS_TOKEN_TTL_MS = 15 * 60_000;
export const REFRESH_TOKEN_TTL_MS = 7 * 24 * 60 * 60_000;

const UNAUTHORIZED = failure(401, 'Not signed in.');

export function createStubAuthRepo(store: StubStore, clock: Clock): AuthRepo {
	function issueAccess(accountId: string, refreshToken: string) {
		const accessToken = `stub-access-${crypto.randomUUID()}`;
		const accessExpiresAt = new Date(clock.now().getTime() + ACCESS_TOKEN_TTL_MS);
		store.accessTokens.set(accessToken, { accountId, expiresAt: accessExpiresAt, refreshToken });
		return { accessToken, accessExpiresAt };
	}

	function issueTokens(accountId: string): TokenPair {
		const refreshToken = `stub-refresh-${crypto.randomUUID()}`;
		const refreshExpiresAt = new Date(clock.now().getTime() + REFRESH_TOKEN_TTL_MS);
		store.refreshTokens.set(refreshToken, { accountId, expiresAt: refreshExpiresAt });
		return { ...issueAccess(accountId, refreshToken), refreshToken, refreshExpiresAt };
	}

	const expired = (expiresAt: Date) => expiresAt.getTime() <= clock.now().getTime();

	return {
		async login({ email, password }) {
			const simulated = simulatedFailure(email);
			if (simulated) return simulated;

			const account = findAccountByEmail(store, email);
			if (!account || account.password !== password) return failure(401, 'Wrong credentials.');
			return ok(issueTokens(account.id));
		},

		async signup({ name, email, password }) {
			const simulated = simulatedFailure(email);
			if (simulated) return simulated;
			if (findAccountByEmail(store, email)) return failure(409, 'Email taken.');

			// Open signup for now: every account gets its own tenant. Invites come later.
			store.sequence += 1;
			const account: StubAccount = {
				id: `acc-${store.sequence}`,
				tenantId: `t-${store.sequence}`,
				profile: { name, email, organization: 'Independent', role: 'admin', title: '' },
				password,
				passwordChangedAt: clock.now()
			};
			store.accounts.set(account.id, account);
			return ok(issueTokens(account.id));
		},

		async refresh(refreshToken) {
			const token = store.refreshTokens.get(refreshToken);
			if (!token || expired(token.expiresAt)) {
				store.refreshTokens.delete(refreshToken);
				return UNAUTHORIZED;
			}
			// The refresh token is kept, not rotated, so parallel requests that
			// refresh at the same moment don't sign each other out.
			return ok({
				...issueAccess(token.accountId, refreshToken),
				refreshToken,
				refreshExpiresAt: token.expiresAt
			});
		},

		async logout(refreshToken) {
			store.refreshTokens.delete(refreshToken);
			for (const [accessToken, token] of store.accessTokens) {
				if (token.refreshToken === refreshToken) store.accessTokens.delete(accessToken);
			}
			return ok(undefined);
		},

		async getSession(accessToken) {
			const token = store.accessTokens.get(accessToken);
			const account = token && store.accounts.get(token.accountId);
			if (!token || !account || expired(token.expiresAt)) {
				store.accessTokens.delete(accessToken);
				return UNAUTHORIZED;
			}
			return ok({
				scope: { tenantId: account.tenantId, accountId: account.id },
				name: account.profile.name
			});
		}
	};
}
