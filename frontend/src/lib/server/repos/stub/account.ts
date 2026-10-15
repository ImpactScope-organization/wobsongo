import type { Clock } from '#lib/utils/clock.js';
import { failure, ok, type Result } from '../../result.js';
import type { Scope } from '../../scope.js';
import type { AccountRepo } from '../../services/account.js';
import { findAccountByEmail, simulatedFailure, type StubAccount, type StubStore } from './store.js';

export function createStubAccountRepo(store: StubStore, clock: Clock): AccountRepo {
	function accountFor(scope: Scope): Result<StubAccount> {
		const simulated = simulatedFailure(scope.accountId);
		if (simulated) return simulated;
		const account = store.accounts.get(scope.accountId);
		return account ? ok(account) : failure(404, 'Account not found.');
	}

	return {
		async getSettings(scope) {
			const result = accountFor(scope);
			if (!result.ok) return result;
			return ok(
				structuredClone({
					profile: result.data.profile,
					passwordChangedAt: result.data.passwordChangedAt
				})
			);
		},

		async updateProfile(scope, { name, email }) {
			const result = accountFor(scope);
			if (!result.ok) return result;
			const simulated = simulatedFailure(email);
			if (simulated) return simulated;

			const owner = findAccountByEmail(store, email);
			if (owner && owner.id !== scope.accountId) return failure(409, 'Email taken.');

			const account = result.data;
			account.profile = { ...account.profile, name, email };
			return ok(structuredClone(account.profile));
		},

		async changePassword(scope, { currentPassword, newPassword }) {
			const result = accountFor(scope);
			if (!result.ok) return result;

			const account = result.data;
			if (account.password !== currentPassword) return failure(403, 'Wrong password.');
			account.password = newPassword;
			account.passwordChangedAt = clock.now();
			return ok(undefined);
		},

		async requestPasswordReset(scope) {
			const result = accountFor(scope);
			if (!result.ok) return result;
			// Nothing is sent; the stub only says where it would go.
			return ok({ sentTo: result.data.profile.email });
		}
	};
}
