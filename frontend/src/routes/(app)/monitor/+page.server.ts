import { error, fail, redirect } from '@sveltejs/kit';
import { parseTimeRange } from '#lib/domain/time-range.js';
import { requireSession } from '#lib/server/auth/session.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, url }) => {
	const session = requireSession(locals, url);
	const range = parseTimeRange(url.searchParams.get('range'));

	const result = await locals.services.monitor.getMonitor(session.scope, { range });
	if (!result.ok) error(result.status, result.message);

	return { snapshot: result.data, range, selectedRunId: url.searchParams.get('run') };
};

export const actions = {
	// Starts a manual run, then shows it selected, keeping the time range.
	run: async ({ locals, url }) => {
		const session = requireSession(locals, url);
		const result = await locals.services.monitor.startRun(session.scope, {
			executedBy: session.name
		});
		if (!result.ok) return fail(result.status, { error: result.message });

		const next = new URLSearchParams({ run: result.data.id });
		const range = url.searchParams.get('range');
		if (range) next.set('range', parseTimeRange(range));
		redirect(303, `${url.pathname}?${next}`);
	}
} satisfies Actions;
