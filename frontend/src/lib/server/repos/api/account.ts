import { notImplemented } from '../../result.js';
import type { AccountRepo } from '../../services/account.js';
import type { ApiOptions } from './index.js';

export function createApiAccountRepo(_options: ApiOptions): AccountRepo {
	return {
		// TODO(openapi): GET /api/v1/me/settings → {name, email, organization, role, title,
		// password_changed_at}.
		async getSettings() {
			return notImplemented('Settings');
		},

		// TODO(openapi): PATCH /api/v1/me {name, email} → profile. 409 if the email is taken.
		async updateProfile() {
			return notImplemented('Saving your profile');
		},

		// TODO(openapi): PUT /api/v1/me/password {current_password, new_password}. 403 if the
		// current password is wrong.
		async changePassword() {
			return notImplemented('Changing your password');
		},

		// TODO(openapi): POST /api/v1/me/password/reset → {sent_to}.
		async requestPasswordReset() {
			return notImplemented('Password reset');
		}
	};
}
