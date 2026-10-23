import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import type { OverviewTopic } from '#lib/domain/overview.js';
import type { TimeRange } from '#lib/domain/time-range.js';
import {
	highHarmSummary,
	narratives,
	platformCounts,
	overviewSnapshot,
	sampleCreators
} from '#lib/fixtures/overview.js';
import { withLocale } from '#lib/testing/locale.js';
import { stripComments } from '#lib/testing/snippet.js';
import CreatorList from './creator-list.svelte';
import HighHarmSummary from './high-harm-summary.svelte';
import NarrativeList from './narrative-list.svelte';
import PlatformBreakdown from './platform-breakdown.svelte';
import OverviewView from './overview-view.svelte';

describe('NarrativeList', () => {
	it('renders an ordered list in the given order', () => {
		const { body } = render(NarrativeList, { props: { narratives } });
		expect(stripComments(body)).toMatch(/^<ol aria-label="Narratives"/);
		const titles = [
			...body.matchAll(/<a href="\/narratives\/[^"]+"[^>]*>\s*([^<]+?)\s*<\/a>/g)
		].map((match) => match[1]);
		expect(titles).toEqual(narratives.map((narrative) => narrative.title));
	});

	it('shows verdict, status, harm, volume and growth for each narrative', () => {
		const { body } = render(NarrativeList, { props: { narratives: narratives.slice(0, 1) } });
		expect(body).toContain('data-verdict="false"');
		expect(body).toContain('data-status="needs-check"');
		expect(body).toContain('Harm 4/5');
		expect(body).toMatch(/1,284<span class="sr-only">(&nbsp;|\u00a0)messages<\/span>/);
		expect(body).toContain('+412%');
	});

	it('highlights spiking growth only', () => {
		const spiking = render(NarrativeList, { props: { narratives: [narratives[0]] } }).body;
		const steady = render(NarrativeList, { props: { narratives: [narratives[1]] } }).body;
		expect(spiking).toContain('data-spiking="true"');
		expect(spiking).toContain('Volume trend, spiking');
		expect(steady).not.toContain('data-spiking=');
	});

	it('builds links with hrefFor', () => {
		const { body } = render(NarrativeList, {
			props: { narratives: [narratives[0]], hrefFor: (id: string) => `/n?id=${id}` }
		});
		expect(body).toContain('href="/n?id=N-0412"');
	});

	it('shows an empty state', () => {
		const { body } = render(NarrativeList, { props: { narratives: [] } });
		expect(body).toContain('No narratives in this window.');
	});
});

describe('CreatorList', () => {
	it('renders creators with numbering, names, and handles without avatars', () => {
		const { body } = render(CreatorList, { props: { creators: sampleCreators } });
		expect(body).toContain('aria-label="Content creators"');
		expect(body).toContain('Ibrahim Cissé');
		expect(body).toContain('@ibrahim_cisse');
		expect(body).toContain('Aïcha Ouédraogo');
		expect(body).toContain('@aicha_oued');
		expect(body).not.toContain('<img');
	});

	it('renders platform icons for tiktok, instagram, facebook, and x', () => {
		const { body } = render(CreatorList, { props: { creators: sampleCreators } });
		expect(body).toContain('title="tiktok"');
		expect(body).toContain('title="facebook"');
		expect(body).toContain('title="x"');
		expect(body).toContain('title="instagram"');
	});

	it('displays formatted followers and posts counts', () => {
		const { body } = render(CreatorList, { props: { creators: sampleCreators.slice(0, 1) } });
		expect(body).toContain('482K');
		expect(body).toContain('followers');
		expect(body).toContain('38');
		expect(body).toContain('posts');
	});

	it('displays rated false or misleading count with verdict meter', () => {
		const { body } = render(CreatorList, { props: { creators: sampleCreators.slice(0, 1) } });
		// 18 false + 12 misleading = 30 out of 38 total (18+12+5+3=38)
		expect(body).toContain('30');
		expect(body).toContain('/ 38');
		expect(body).toContain('role="img"');
	});

	it('shows an empty state when no creators are present', () => {
		const { body } = render(CreatorList, { props: { creators: [] } });
		expect(body).toContain('No content creators detected in this window.');
	});

	it('ranks creators by volume (posts/claims count) highest first', () => {
		const { body } = render(CreatorList, { props: { creators: sampleCreators } });
		// Aïcha Ouédraogo has 52 posts (highest), Ibrahim Cissé has 38 posts
		const aichaIndex = body.indexOf('Aïcha Ouédraogo');
		const ibrahimIndex = body.indexOf('Ibrahim Cissé');
		expect(aichaIndex).toBeLessThan(ibrahimIndex);
	});

	it('swaps creator names with pseudonyms, hides handles, and masks metrics when privacyMode is enabled', () => {
		const { body } = render(CreatorList, {
			props: { creators: sampleCreators, privacyMode: true }
		});
		expect(body).not.toContain('Aïcha Ouédraogo');
		expect(body).not.toContain('Ibrahim Cissé');
		expect(body).not.toContain('@aicha_oued');
		expect(body).not.toContain('@ibrahim_cisse');
		expect(body).toContain('Creator');
		expect(body).toContain('Creator 01');
		expect(body).toContain('Creator 03');
		// Followers and posts count are masked to "-" in privacy mode
		expect(body).not.toContain('890K');
		expect(body).not.toContain('482K');
		expect(body).toContain('-');
		// Rated false/misleading is shown as percentage
		expect(body).toContain('77%');
		expect(body).not.toContain('/ 52');
		expect(body).toContain('role="img"');
	});

	it('swaps creator names with pseudonyms and hides handles when hasAccess is false', () => {
		const { body } = render(CreatorList, {
			props: { creators: sampleCreators, hasAccess: false }
		});
		expect(body).not.toContain('Aïcha Ouédraogo');
		expect(body).not.toContain('@aicha_oued');
		expect(body).toContain('Creator 01');
		expect(body).not.toContain('890K');
		expect(body).toContain('-');
		expect(body).toContain('77%');
		expect(body).not.toContain('/ 52');
	});

	it('renders risk level column with radial gauges representing 0-100 risk score', () => {
		const { body } = render(CreatorList, { props: { creators: sampleCreators } });
		expect(body).toContain('Risk level');
		expect(body).toContain('role="meter"');
		// High risk (red tier, e.g. 82)
		expect(body).toContain('aria-valuenow="82"');
		expect(body).toContain('82');
		// Medium risk (orange tier, e.g. 58)
		expect(body).toContain('aria-valuenow="58"');
		expect(body).toContain('58');
		// Low risk (green tier, e.g. 28)
		expect(body).toContain('aria-valuenow="28"');
		expect(body).toContain('28');
	});

	it('makes list items a link to profileUrl when !isPrivate', () => {
		const { body } = render(CreatorList, { props: { creators: sampleCreators } });
		expect(body).toContain('href="https://www.tiktok.com/@ibrahim_cisse"');
		expect(body).toContain('target="_blank"');
		expect(body).toContain('rel="noopener noreferrer"');
	});

	it('supports a custom profileUrl resolver function', () => {
		const { body } = render(CreatorList, {
			props: {
				creators: sampleCreators,
				profileUrl: (creator) => `/creators/${creator.id}`
			}
		});
		expect(body).toContain('href="/creators/c1"');
		expect(body).toContain('href="/creators/c2"');
	});

	it('renders a regular list with no link or redirect when privacyMode is enabled or hasAccess=false', () => {
		const privateView = render(CreatorList, {
			props: { creators: sampleCreators, privacyMode: true }
		}).body;
		expect(privateView).not.toContain('<a href');
		expect(privateView).not.toContain('https://www.tiktok.com');

		const noAccessView = render(CreatorList, {
			props: { creators: sampleCreators, hasAccess: false }
		}).body;
		expect(noAccessView).not.toContain('<a href');
		expect(noAccessView).not.toContain('https://www.tiktok.com');
	});
});

describe('PlatformBreakdown', () => {
	it('draws one donut slice per platform on the server', () => {
		const { body } = render(PlatformBreakdown, { props: { counts: platformCounts } });
		expect(body.match(/<path[^>]*class="[^"]*lc-arc/g)).toHaveLength(platformCounts.length);
	});

	it('lists counts and shares that add up to 100%', () => {
		const { body } = render(PlatformBreakdown, { props: { counts: platformCounts } });
		expect(body).toContain('WhatsApp (bot)');
		expect(body).toContain('2,693');
		expect(body).toContain('5,788');
		const shares = [...body.matchAll(/>(\d+)%<\/td>/g)].map((match) => Number(match[1]));
		expect(shares).toEqual([46, 27, 19, 5, 3]);
	});

	it('colours the table swatches with the platform colours', () => {
		const { body } = render(PlatformBreakdown, { props: { counts: platformCounts } });
		// The --color-* variables only exist inside the chart container, not the table.
		expect(body).toContain('background: var(--platform-whatsapp)');
		expect(body).not.toContain('var(--color-');
	});

	it('describes an empty chart without a dangling colon', () => {
		const { body } = render(PlatformBreakdown, { props: { counts: [] } });
		expect(body).toContain('aria-label="No claims by platform yet"');
	});

	it('describes the chart for screen readers', () => {
		const { body } = render(PlatformBreakdown, { props: { counts: platformCounts } });
		expect(body).toContain('aria-label="Claims by platform: WhatsApp (bot) 46%, Facebook 27%');
	});

	it('draws no slices when there is no data', () => {
		const counts = platformCounts.map((item) => ({ ...item, count: 0 }));
		const { body } = render(PlatformBreakdown, { props: { counts } });
		expect(body).not.toContain('lc-arc');
		expect(body).toContain('>0%</td>');
	});
});

describe('HighHarmSummary', () => {
	it('links the whole card to the high-harm view', () => {
		const { body } = render(HighHarmSummary, {
			props: { summary: highHarmSummary, href: '/safety' }
		});
		expect(body).toMatch(/<h2[^>]*>\s*<a href="\/safety"/);
	});

	it('pluralises the counts', () => {
		const { body } = render(HighHarmSummary, {
			props: { summary: highHarmSummary, href: '/safety' }
		});
		expect(body).toContain('dangerous-practice claims awaiting expert review');
		expect(body).toContain('claim fast-tracked to the fact-check queue');
	});
});

describe('OverviewView', () => {
	const hrefs = {
		rangeHref: (range: string) => `?range=${range}&topic=misinfo`,
		topicHref: (topic: string) => `?range=24h&topic=${topic}`
	};
	const view = (
		snapshot = overviewSnapshot,
		range: TimeRange = '24h',
		topic: OverviewTopic = 'misinfo'
	) => stripComments(render(OverviewView, { props: { snapshot, range, topic, ...hrefs } }).body);

	it('shows the headline numbers with changes against the previous window', () => {
		const body = view();
		expect(body).toContain('Burkina Faso · updated 14:05 GMT');
		expect(body).toContain('Messages received, 24 h');
		expect(body).toContain('8,412');
		expect(body).toContain('▲ 64% vs prev. 24 h');
		expect(body).toContain('6 new in the last 24 h');
		expect(body).toMatch(/>21<span[^>]*>(&nbsp;|\u00a0)\/ 37</);
		expect(body).toContain('▲ 18 min vs prev. 24 h');
	});

	it('says when things fell or held steady', () => {
		const body = view(
			{
				...overviewSnapshot,
				messages: { ...overviewSnapshot.messages, previous: overviewSnapshot.messages.current },
				medianDebunkMinutes: { current: 100, previous: 120 }
			},
			'7d'
		);
		expect(body).toContain('No change vs prev. 7 d');
		expect(body).toContain('▼ 20 min vs prev. 7 d');
	});

	it('links the range and topic options, marking the current ones', () => {
		const body = view(overviewSnapshot, '7d', 'all');
		expect(body).toMatch(/<a href="\?range=7d&amp;topic=misinfo" aria-current="true"/);
		expect(body).toMatch(/<a href="\?range=24h&amp;topic=all" aria-current="true"/);
		expect(body).toContain('Top narratives');
	});

	it('shows the anomaly banner only when there is one', () => {
		expect(view()).toContain('aria-label="Anomaly"');
		expect(view({ ...overviewSnapshot, anomaly: null })).not.toContain('aria-label="Anomaly"');
	});

	it('lists the narratives it is given, in order', () => {
		const body = view();
		const first = body.indexOf(narratives[0].title);
		const second = body.indexOf(narratives[1].title);
		expect(first).toBeGreaterThan(-1);
		expect(second).toBeGreaterThan(first);
		expect(view({ ...overviewSnapshot, narratives: [] })).toContain(
			'No narratives in this window.'
		);
	});
});

describe('Overview in French', () => {
	it('words the headline numbers, sections and lists in French', () => {
		const body = withLocale('fr', () =>
			stripComments(
				render(OverviewView, {
					props: {
						snapshot: overviewSnapshot,
						range: '7d',
						topic: 'misinfo',
						rangeHref: (range: string) => `?range=${range}`,
						topicHref: (topic: string) => `?topic=${topic}`
					}
				}).body
			)
		);
		expect(body).toContain('Burkina Faso · mis à jour à 14:05 GMT');
		expect(body).toContain('Messages reçus, 7 j');
		expect(body).toContain('8\u202f412');
		expect(body).toContain('vs 7 j préc.');
		expect(body).toContain('6 nouveaux en 7 j');
		expect(body).toContain('Principaux récits de désinformation');
		expect(body).toContain('D’où viennent les affirmations');
		expect(body).toContain('Affirmations à risque élevé');
		expect(body).toContain('aria-label="Récits"');
	});

	it('counts one claim in the singular', () => {
		const body = withLocale(
			'fr',
			() =>
				render(HighHarmSummary, {
					props: { summary: { awaitingReview: 1, fastTracked: 0 }, href: '/safety' }
				}).body
		);
		expect(body).toContain('affirmation sur une pratique dangereuse en attente');
		// French counts 0 as singular too.
		expect(body).toContain('affirmation traitée en priorité');
	});
});
