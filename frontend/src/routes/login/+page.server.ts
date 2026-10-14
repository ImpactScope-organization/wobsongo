import { fail } from '@sveltejs/kit';
import type { Actions } from './$types';

export const actions = {
	// Email/password sign-in is wired up with the auth service (tr.md, "Authentication").
	// Until then the form says so instead of failing with a 405.
	default: async ({ request }) => {
		const data = await request.formData();
		const email = String(data.get('email') ?? '');
		return fail(501, { email, message: 'Logging in is not available yet.' });
	}
} satisfies Actions;
