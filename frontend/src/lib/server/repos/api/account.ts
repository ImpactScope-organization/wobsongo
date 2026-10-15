import { notImplemented } from '../../result.js';
import type { AccountRepo } from '../../services/account.js';
import { call, createApiClient, toTokenPair, type Schemas } from './client.js';
import type { ApiOptions } from './index.js';

function toProfile(me: Schemas['dto.MeResponse']) {
	return { name: me.name, email: me.email, role: me.role };
}

// The bearer token comes from `handleFetch`; the backend knows who is asking,
// so the scope isn't sent.
export function createApiAccountRepo(options: ApiOptions): AccountRepo {
	const api = createApiClient(options);
	return {
		getSettings: () =>
			call(api.GET('/api/v1/me'), (me) => ({
				profile: toProfile(me),
				passwordChangedAt: new Date(me.password_changed_at)
			})),

		updateProfile: (_scope, change) => call(api.PATCH('/api/v1/me', { body: change }), toProfile),

		changePassword: (_scope, { currentPassword, newPassword }) =>
			call(
				api.POST('/api/v1/me/password', {
					body: { current_password: currentPassword, new_password: newPassword }
				}),
				toTokenPair
			),

		// TODO: needs a mailer on the backend first.
		async requestPasswordReset() {
			return notImplemented('Password reset');
		}
	};
}
