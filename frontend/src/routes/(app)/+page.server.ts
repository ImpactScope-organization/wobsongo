import { error } from '@sveltejs/kit';
import { parseOverviewTopic } from '#lib/domain/overview.js';
import { parseTimeRange } from '#lib/domain/time-range.js';
import { requireSession } from '#lib/server/auth/session.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, url }) => {
	const session = requireSession(locals, url);
	const range = parseTimeRange(url.searchParams.get('range'));
	const topic = parseOverviewTopic(url.searchParams.get('topic'));

	const result = await locals.services.overview.getOverview(session.scope, { range, topic });
	if (!result.ok) error(result.status, result.message);

	return { snapshot: result.data, range, topic };
};
