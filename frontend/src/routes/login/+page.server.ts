import { fail, redirect } from '@sveltejs/kit';
import { setTokenCookies } from '#lib/server/auth/cookies.js';
import { safeRedirectTarget } from '#lib/server/auth/session.js';
import { formText } from '#lib/server/forms.js';
import type { Actions } from './$types';

export const actions = {
	default: async ({ cookies, locals, request, url }) => {
		const data = await request.formData();
		const email = formText(data, 'email');

		const result = await locals.services.auth.login({
			email,
			password: formText(data, 'password')
		});
		if (!result.ok) return fail(result.status, { email, message: result.message });

		setTokenCookies(cookies, result.data, locals.clock.now());
		redirect(303, safeRedirectTarget(url.searchParams.get('redirectTo')));
	}
} satisfies Actions;
