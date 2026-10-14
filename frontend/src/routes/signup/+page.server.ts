import { fail } from '@sveltejs/kit';
import type { Actions } from './$types';

export const actions = {
	// Account creation is wired up with the auth service (tr.md, "Authentication").
	// Until then the form says so instead of failing with a 405.
	default: async ({ request }) => {
		const data = await request.formData();
		const values = { name: String(data.get('name') ?? ''), email: String(data.get('email') ?? '') };
		return fail(501, { values, message: 'Signing up is not available yet.' });
	}
} satisfies Actions;
