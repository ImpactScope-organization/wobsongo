import { error } from '@sveltejs/kit';
import { comingSoonSection } from '#lib/navigation.js';
import type { PageServerLoad } from './$types';

// Views in the nav that aren't built yet (and links into them, e.g. a
// narrative from Pulse) say so; any other unknown path is a 404.
export const load: PageServerLoad = ({ url }) => {
	const section = comingSoonSection(url.pathname);
	if (!section) error(404, 'Not found');
	return { title: section.label };
};
