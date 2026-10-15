import { error, fail } from '@sveltejs/kit';
import { requireSession } from '#lib/server/auth/session.js';
import { formText } from '#lib/server/forms.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, url }) => {
	const session = requireSession(locals, url);
	const result = await locals.services.account.getSettings(session.scope);
	if (!result.ok) error(result.status, result.message);

	return { ...result.data, now: locals.clock.now() };
};

// `form` says which section a result belongs to, so only that one shows it.
export const actions = {
	profile: async ({ locals, request, url }) => {
		const session = requireSession(locals, url);
		const data = await request.formData();
		const values = { name: formText(data, 'name'), email: formText(data, 'email') };

		const result = await locals.services.account.updateProfile(session.scope, values);
		if (!result.ok)
			return fail(result.status, { form: 'profile' as const, values, error: result.message });
		return { form: 'profile' as const, saved: true };
	},

	// Passwords are never sent back to the page, not even after a failure.
	password: async ({ locals, request, url }) => {
		const session = requireSession(locals, url);
		const data = await request.formData();

		const result = await locals.services.account.changePassword(session.scope, {
			currentPassword: formText(data, 'currentPassword'),
			newPassword: formText(data, 'newPassword'),
			confirmPassword: formText(data, 'confirmPassword')
		});
		if (!result.ok)
			return fail(result.status, { form: 'password' as const, error: result.message });
		return { form: 'password' as const, updated: true };
	},

	resetPassword: async ({ locals, url }) => {
		const session = requireSession(locals, url);
		const result = await locals.services.account.requestPasswordReset(session.scope);
		if (!result.ok)
			return fail(result.status, { form: 'password' as const, error: result.message });
		return { form: 'password' as const, resetSentTo: result.data.sentTo };
	}
} satisfies Actions;
