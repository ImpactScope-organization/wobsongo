import { redirect } from '@sveltejs/kit';
import { cookieMaxAge, cookieName, isLocale } from '#lib/paraglide/runtime.js';
import { safeRedirectTarget } from '#lib/server/auth/session.js';
import { formText } from '#lib/server/forms.js';
import type { RequestHandler } from './$types';

// The language switcher posts here. The cookie is the one Paraglide reads on
// both sides, so it can't be httpOnly. Unknown languages leave it as it was.
export const POST: RequestHandler = async ({ cookies, request }) => {
	const data = await request.formData();
	const lang = formText(data, 'lang');
	if (isLocale(lang)) {
		cookies.set(cookieName, lang, {
			path: '/',
			httpOnly: false,
			sameSite: 'lax',
			maxAge: cookieMaxAge
		});
	}
	redirect(303, safeRedirectTarget(formText(data, 'redirectTo')));
};
