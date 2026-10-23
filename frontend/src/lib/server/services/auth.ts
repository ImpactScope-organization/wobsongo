import { newPasswordError, normalizeEmail, profileError } from '#lib/domain/account.js';
import { m } from '#lib/paraglide/messages.js';
import { failure, type Result } from '../result.js';
import type { Scope } from '../scope.js';
import { describeFailure } from './failures.js';

/** Tokens issued at sign-in. Both live in httpOnly cookies; the server sends them on. */
export interface TokenPair {
	accessToken: string;
	accessExpiresAt: Date;
	refreshToken: string;
	refreshExpiresAt: Date;
}

/** Who is signed in. */
export interface Session {
	scope: Scope;
	name: string;
}

export interface Credentials {
	email: string;
	password: string;
}

export interface Signup {
	name: string;
	email: string;
	password: string;
	confirmPassword: string;
}

/** Where sessions come from: the backend, or a stub. Input is already validated. */
export interface AuthRepo {
	/** 401 when the email or password is wrong. */
	login(credentials: Credentials): Promise<Result<TokenPair>>;
	/** Creates the account and signs it in. 409 when the email is taken. */
	signup(account: { name: string; email: string; password: string }): Promise<Result<TokenPair>>;
	/** 401 when the refresh token is expired or revoked. */
	refresh(refreshToken: string): Promise<Result<TokenPair>>;
	/**
	 * Ends the session where the backend can. Its tokens are stateless, so
	 * there it does nothing; signing out is clearing the cookies.
	 */
	logout(refreshToken: string): Promise<Result<void>>;
	/** 401 when the access token is expired or unknown. */
	getSession(accessToken: string): Promise<Result<Session>>;
}

export type AuthService = Omit<AuthRepo, 'signup'> & {
	signup(account: Signup): Promise<Result<TokenPair>>;
};

// Same message for an unknown email and a wrong password, so the form can't
// be used to find out who has an account.

export function createAuthService(repo: AuthRepo): AuthService {
	return {
		async login({ email, password }) {
			if (!email.trim() || !password) return failure(400, m.auth_credentials_required());
			const result = await repo.login({ email: normalizeEmail(email), password });
			return result.ok
				? result
				: describeFailure(result, {
						401: m.auth_bad_credentials(),
						429: m.auth_too_many_attempts()
					});
		},

		async signup(account) {
			const invalid =
				profileError(account) ??
				newPasswordError({
					newPassword: account.password,
					confirmPassword: account.confirmPassword
				});
			if (invalid) return failure(400, invalid);

			const result = await repo.signup({
				name: account.name.trim(),
				email: normalizeEmail(account.email),
				password: account.password
			});
			return result.ok
				? result
				: describeFailure(result, {
						409: m.auth_email_taken(),
						429: m.auth_too_many_attempts()
					});
		},

		refresh: (refreshToken) => repo.refresh(refreshToken),
		logout: (refreshToken) => repo.logout(refreshToken),
		getSession: (accessToken) => repo.getSession(accessToken)
	};
}
