import type { Clock } from '#lib/utils/clock.js';
import { failure, ok } from '../../result.js';
import type { AuthRepo } from '../../services/auth.js';
import { findAccountByEmail, simulatedFailure, type StubAccount, type StubStore } from './store.js';
import { issueAccess, issueTokens } from './tokens.js';

const UNAUTHORIZED = failure(401, 'Not signed in.');

export function createStubAuthRepo(store: StubStore, clock: Clock): AuthRepo {
	const expired = (expiresAt: Date) => expiresAt.getTime() <= clock.now().getTime();

	return {
		async login({ email, password }) {
			const simulated = simulatedFailure(email);
			if (simulated) return simulated;

			const account = findAccountByEmail(store, email);
			if (!account || account.password !== password) return failure(401, 'Wrong credentials.');
			return ok(issueTokens(store, clock, account.id));
		},

		async signup({ name, email, password }) {
			const simulated = simulatedFailure(email);
			if (simulated) return simulated;
			if (findAccountByEmail(store, email)) return failure(409, 'Email taken.');

			store.sequence += 1;
			const account: StubAccount = {
				id: `acc-${store.sequence}`,
				profile: { name, email, role: 'user' },
				password,
				passwordChangedAt: clock.now()
			};
			store.accounts.set(account.id, account);
			return ok(issueTokens(store, clock, account.id));
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
				...issueAccess(store, clock, token.accountId, refreshToken),
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
				scope: { accountId: account.id },
				name: account.profile.name
			});
		}
	};
}
