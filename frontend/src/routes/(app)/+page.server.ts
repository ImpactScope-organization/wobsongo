import { error } from '@sveltejs/kit';
import { PRIVACY_WHITELIST_EMAILS } from '$app/env/private';
import { parseOverviewTopic } from '#lib/domain/overview.js';
import { parseTimeRange } from '#lib/domain/time-range.js';
import { requireSession } from '#lib/server/auth/session.js';
import { isWhitelistedForFullData } from '#lib/server/privacy.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, url }) => {
	const session = requireSession(locals, url);
	const range = parseTimeRange(url.searchParams.get('range'));
	const topic = parseOverviewTopic(url.searchParams.get('topic'));

	// A logged in user has access to the overview
	const hasAccess = url.searchParams.get('hasAccess') !== 'false';

	// Full data access is restricted to emails on the whitelist
	let userEmail = (session as { email?: string }).email;
	if (!userEmail) {
		const account = await locals.services.account.getSettings(session.scope);
		if (account.ok) userEmail = account.data.profile.email;
	}

	const whitelistStr = process.env.PRIVACY_WHITELIST_EMAILS || PRIVACY_WHITELIST_EMAILS || '';
	const hasFullAccess = isWhitelistedForFullData(userEmail, whitelistStr);

	// Whitelisted users automatically see full data without needing to toggle,
	// but can optionally toggle ?privacy=true.
	// Non-whitelisted users always see private data.
	const privacyParam = url.searchParams.get('privacy') ?? url.searchParams.get('privacyMode');
	const privacyMode = hasFullAccess ? privacyParam === 'true' : true;
	const isPrivate = !hasAccess || privacyMode;

	const result = await locals.services.overview.getOverview(session.scope, { range, topic });
	if (!result.ok) error(result.status, result.message);

	const snapshot = isPrivate
		? {
				...result.data,
				creators: result.data.creators?.map(
					({ followers: _f, postsCount: _p, handle: _h, profileUrl: _url, ...creator }, index) => ({
						...creator,
						name: creator.pseudoname || creator.pseudonym || `Creator ${index + 1}`
					})
				)
			}
		: result.data;

	return { snapshot, range, topic, privacyMode, hasAccess };
};
