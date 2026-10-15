import { notImplemented } from '../../result.js';
import type { AuthRepo } from '../../services/auth.js';
import type { ApiOptions } from './index.js';

export function createApiAuthRepo(_options: ApiOptions): AuthRepo {
	return {
		// TODO(openapi): POST /api/v1/auth/login {email, password} → {access_token, access_expires_at,
		// refresh_token, refresh_expires_at}. 401 on bad credentials.
		async login() {
			return notImplemented('Logging in');
		},

		// TODO(openapi): POST /api/v1/auth/signup {name, email, password} → token pair. 409 if the
		// email is taken. Open signup creates a tenant per account; switch to invites later.
		async signup() {
			return notImplemented('Signing up');
		},

		// TODO(openapi): POST /api/v1/auth/refresh {refresh_token} → token pair. 401 if expired or revoked.
		async refresh() {
			return notImplemented('Staying signed in');
		},

		// TODO(openapi): POST /api/v1/auth/logout {refresh_token}; revokes it.
		async logout() {
			return notImplemented('Logging out');
		},

		// TODO(openapi): GET /api/v1/me with the bearer token → {account_id, tenant_id, name}, or
		// decode the JWT's claims (the backend verifies it on every call). 401 if invalid. The
		// tenant is the owner's when the account is shared (tr.md, "service layer").
		async getSession() {
			return notImplemented('Signing in');
		}
	};
}
