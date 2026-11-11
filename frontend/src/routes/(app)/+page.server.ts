import { error } from '@sveltejs/kit';
import { PRIVACY_WHITELIST_EMAILS } from '$app/env/private';
import { parseOverviewTopic, parseEngagementWindow } from '#lib/domain/overview.js';
import { parseTimeRange } from '#lib/domain/time-range.js';
import { m } from '#lib/paraglide/messages.js';
import { requireSession } from '#lib/server/auth/session.js';
import { isWhitelistedForFullData } from '#lib/server/privacy.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, url }) => {
	const session = requireSession(locals, url);
	const range = parseTimeRange(url.searchParams.get('range'));
	const topic = parseOverviewTopic(url.searchParams.get('topic'));
	const engagement = parseEngagementWindow(url.searchParams.get('engagement'));
	// A logged in user has access to the overview
	const hasAccess = url.searchParams.get('hasAccess') !== 'false';

	// Full data access is restricted to emails on the whitelist. If the email
	// can't be read, no one counts as whitelisted: creators stay masked.
	const whitelistStr = process.env.PRIVACY_WHITELIST_EMAILS || PRIVACY_WHITELIST_EMAILS || '';
	let hasFullAccess = false;
	const account = await locals.services.account.getSettings(session.scope);
	if (!account.ok) {
		console.error('[overview] account unreadable, showing masked data (status %d)', account.status);
	} else {
		hasFullAccess = isWhitelistedForFullData(account.data.profile.email, whitelistStr);
	}

	// Whitelisted users automatically see full data without needing to toggle,
	// but can optionally toggle ?privacy=true.
	// Non-whitelisted users always see private data.
	const privacyParam = url.searchParams.get('privacy') ?? url.searchParams.get('privacyMode');
	const privacyMode = hasFullAccess ? privacyParam === 'true' : true;
	const isPrivate = !hasAccess || privacyMode;

	const result = await locals.services.overview.getOverview(session.scope, {
		range,
		topic,
		engagement
	});
	if (!result.ok) error(result.status, result.message);

	const snapshot = isPrivate
		? {
				...result.data,
				creators: result.data.creators.map(
					(
						{
							followers: _f,
							postsCount: _p,
							handle: _h,
							profileUrl: _url,
							avatarUrl: _a,
							...creator
						},
						index
					) => ({
						...creator,
						name:
							creator.pseudoname ||
							creator.pseudonym ||
							m.overview_creator_pseudonym({ number: index + 1 })
					})
				)
			}
		: result.data;

	return { snapshot, range, topic, engagement, privacyMode, hasAccess };
};
