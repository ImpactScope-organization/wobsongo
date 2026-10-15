import {
	normalizeEmail,
	passwordChangeError,
	profileError,
	type AccountProfile
} from '#lib/domain/account.js';
import { failure, type Result } from '../result.js';
import type { Scope } from '../scope.js';
import type { TokenPair } from './auth.js';
import { describeFailure } from './failures.js';

/** The signed-in person's settings. */
export interface AccountSettings {
	profile: AccountProfile;
	/** When the password last changed; null if never. */
	passwordChangedAt: Date | null;
}

export interface ProfileChange {
	name: string;
	email: string;
}

export interface PasswordChange {
	currentPassword: string;
	newPassword: string;
	confirmPassword: string;
}

/** Where account data comes from: the backend, or a stub. Input is already validated. */
export interface AccountRepo {
	getSettings(scope: Scope): Promise<Result<AccountSettings>>;
	/** 409 when another account uses the email. */
	updateProfile(scope: Scope, change: ProfileChange): Promise<Result<AccountProfile>>;
	/**
	 * 403 when the current password is wrong. Ends the account's other
	 * sessions and returns new tokens for this one.
	 */
	changePassword(
		scope: Scope,
		change: { currentPassword: string; newPassword: string }
	): Promise<Result<TokenPair>>;
	/** Emails a reset link; returns the address it went to. */
	requestPasswordReset(scope: Scope): Promise<Result<{ sentTo: string }>>;
}

export interface AccountService {
	getSettings(scope: Scope): Promise<Result<AccountSettings>>;
	updateProfile(scope: Scope, change: ProfileChange): Promise<Result<AccountProfile>>;
	changePassword(scope: Scope, change: PasswordChange): Promise<Result<TokenPair>>;
	requestPasswordReset(scope: Scope): Promise<Result<{ sentTo: string }>>;
}

export function createAccountService(repo: AccountRepo): AccountService {
	return {
		async getSettings(scope) {
			const result = await repo.getSettings(scope);
			return result.ok ? result : describeFailure(result, { 404: 'Your account was not found.' });
		},

		async updateProfile(scope, change) {
			const invalid = profileError(change);
			if (invalid) return failure(400, invalid);

			const result = await repo.updateProfile(scope, {
				name: change.name.trim(),
				email: normalizeEmail(change.email)
			});
			return result.ok
				? result
				: describeFailure(result, { 409: 'Another account already uses this email.' });
		},

		async changePassword(scope, change) {
			const invalid = passwordChangeError(change);
			if (invalid) return failure(400, invalid);

			const result = await repo.changePassword(scope, {
				currentPassword: change.currentPassword,
				newPassword: change.newPassword
			});
			return result.ok
				? result
				: describeFailure(result, { 403: 'Your current password is incorrect.' });
		},

		async requestPasswordReset(scope) {
			const result = await repo.requestPasswordReset(scope);
			return result.ok ? result : describeFailure(result);
		}
	};
}
