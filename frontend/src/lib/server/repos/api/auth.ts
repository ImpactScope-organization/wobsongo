import { ok } from '../../result.js';
import type { AuthRepo } from '../../services/auth.js';
import { call, createApiClient, toTokenPair } from './client.js';
import type { ApiOptions } from './index.js';

export function createApiAuthRepo(options: ApiOptions): AuthRepo {
	const api = createApiClient(options);
	return {
		login: (credentials) =>
			call(api.POST('/api/v1/auth/login', { body: credentials }), toTokenPair),

		signup: (account) => call(api.POST('/api/v1/auth/signup', { body: account }), toTokenPair),

		refresh: (refreshToken) =>
			call(
				api.POST('/api/v1/auth/refresh', { body: { refresh_token: refreshToken } }),
				toTokenPair
			),

		// The backend's tokens are stateless: there is nothing to revoke.
		async logout() {
			return ok(undefined);
		},

		// The token is passed explicitly: right after a refresh it is newer than
		// the one `handleFetch` would find in the request's cookies.
		getSession: (accessToken) =>
			call(
				api.GET('/api/v1/me', { headers: { authorization: `Bearer ${accessToken}` } }),
				(me) => ({ scope: { accountId: me.id }, name: me.name })
			)
	};
}
