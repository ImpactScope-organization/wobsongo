import { error, redirect } from '@sveltejs/kit';
import { comingSoonSection } from '#lib/navigation.js';
import { m } from '#lib/paraglide/messages.js';
import type { PageServerLoad } from './$types';

// Views in the nav that aren't built yet (and links into them, e.g. a
// narrative from Overview) say so; any other unknown path is a 404.
export const load: PageServerLoad = ({ url }) => {
	if (url.pathname === '/overview' || url.pathname === '/pulse') {
		redirect(307, '/');
	}
	const section = comingSoonSection(url.pathname);
	if (!section) error(404, m.error_not_found());
	return { title: section.label };
};
