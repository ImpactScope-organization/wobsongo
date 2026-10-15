import { newPasswordError, normalizeEmail, profileError } from '#lib/domain/account.js';
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
	/** Creates the account (and, for now, its own tenant) and signs it in. 409 when the email is taken. */
	signup(account: { name: string; email: string; password: string }): Promise<Result<TokenPair>>;
	/** 401 when the refresh token is expired or revoked. */
	refresh(refreshToken: string): Promise<Result<TokenPair>>;
	logout(refreshToken: string): Promise<Result<void>>;
	/** 401 when the access token is expired or unknown. */
	getSession(accessToken: string): Promise<Result<Session>>;
}

export type AuthService = Omit<AuthRepo, 'signup'> & {
	signup(account: Signup): Promise<Result<TokenPair>>;
};

// Same message for an unknown email and a wrong password, so the form can't
// be used to find out who has an account.
const BAD_CREDENTIALS = 'Email or password is incorrect.';

export function createAuthService(repo: AuthRepo): AuthService {
	return {
		async login({ email, password }) {
			if (!email.trim() || !password) return failure(400, 'Enter your email and password.');
			const result = await repo.login({ email: normalizeEmail(email), password });
			return result.ok ? result : describeFailure(result, { 401: BAD_CREDENTIALS });
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
				: describeFailure(result, { 409: 'An account with this email already exists.' });
		},

		refresh: (refreshToken) => repo.refresh(refreshToken),
		logout: (refreshToken) => repo.logout(refreshToken),
		getSession: (accessToken) => repo.getSession(accessToken)
	};
}
