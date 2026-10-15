import { error } from '@sveltejs/kit';
import { parsePulseTopic } from '#lib/domain/pulse.js';
import { parseTimeRange } from '#lib/domain/time-range.js';
import { requireSession } from '#lib/server/auth/session.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, url }) => {
	const session = requireSession(locals, url);
	const range = parseTimeRange(url.searchParams.get('range'));
	const topic = parsePulseTopic(url.searchParams.get('topic'));

	const result = await locals.services.pulse.getPulse(session.scope, { range, topic });
	if (!result.ok) error(result.status, result.message);

	return { snapshot: result.data, range, topic };
};
