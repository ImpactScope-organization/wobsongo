import { fail, redirect } from '@sveltejs/kit';
import { setTokenCookies } from '#lib/server/auth/cookies.js';
import { formText } from '#lib/server/forms.js';
import type { Actions } from './$types';

export const actions = {
	// Open signup: anyone can create an account.
	default: async ({ cookies, locals, request }) => {
		const data = await request.formData();
		const values = { name: formText(data, 'name'), email: formText(data, 'email') };

		const result = await locals.services.auth.signup({
			...values,
			password: formText(data, 'password'),
			confirmPassword: formText(data, 'confirmPassword')
		});
		if (!result.ok) return fail(result.status, { values, message: result.message });

		setTokenCookies(cookies, result.data, locals.clock.now());
		redirect(303, '/');
	}
} satisfies Actions;
