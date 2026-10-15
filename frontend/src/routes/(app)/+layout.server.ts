import { requireSession } from '#lib/server/auth/session.js';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = ({ locals, url }) => {
	const session = requireSession(locals, url);
	return { account: { name: session.name } };
};
