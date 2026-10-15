import { redirect } from '@sveltejs/kit';
import { REFRESH_COOKIE, clearTokenCookies } from '#lib/server/auth/cookies.js';
import type { RequestHandler } from './$types';

// POST only: a link (GET) that signs out could be triggered by any page.
export const POST: RequestHandler = async ({ cookies, locals }) => {
	const refreshToken = cookies.get(REFRESH_COOKIE);
	// Signed out locally even if the backend can't revoke the token right now.
	if (refreshToken) await locals.services.auth.logout(refreshToken);
	clearTokenCookies(cookies);
	redirect(303, '/login');
};
