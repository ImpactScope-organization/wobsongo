import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import type { PulseTopic } from '#lib/domain/pulse.js';
import type { TimeRange } from '#lib/domain/time-range.js';
import { highHarmSummary, narratives, platformCounts, pulseSnapshot } from '#lib/fixtures/pulse.js';
import { stripComments } from '#lib/testing/snippet.js';
import HighHarmSummary from './high-harm-summary.svelte';
import NarrativeList from './narrative-list.svelte';
import PlatformBreakdown from './platform-breakdown.svelte';
import PulseView from './pulse-view.svelte';

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

describe('PulseView', () => {
	const hrefs = {
		rangeHref: (range: string) => `?range=${range}&topic=misinfo`,
		topicHref: (topic: string) => `?range=24h&topic=${topic}`
	};
	const view = (
		snapshot = pulseSnapshot,
		range: TimeRange = '24h',
		topic: PulseTopic = 'misinfo'
	) => stripComments(render(PulseView, { props: { snapshot, range, topic, ...hrefs } }).body);

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
				...pulseSnapshot,
				messages: { ...pulseSnapshot.messages, previous: pulseSnapshot.messages.current },
				medianDebunkMinutes: { current: 100, previous: 120 }
			},
			'7d'
		);
		expect(body).toContain('No change vs prev. 7 d');
		expect(body).toContain('▼ 20 min vs prev. 7 d');
	});

	it('links the range and topic options, marking the current ones', () => {
		const body = view(pulseSnapshot, '7d', 'all');
		expect(body).toMatch(/<a href="\?range=7d&amp;topic=misinfo" aria-current="true"/);
		expect(body).toMatch(/<a href="\?range=24h&amp;topic=all" aria-current="true"/);
		expect(body).toContain('Top narratives');
	});

	it('shows the anomaly banner only when there is one', () => {
		expect(view()).toContain('aria-label="Anomaly"');
		expect(view({ ...pulseSnapshot, anomaly: null })).not.toContain('aria-label="Anomaly"');
	});

	it('lists the narratives it is given, in order', () => {
		const body = view();
		const first = body.indexOf(narratives[0].title);
		const second = body.indexOf(narratives[1].title);
		expect(first).toBeGreaterThan(-1);
		expect(second).toBeGreaterThan(first);
		expect(view({ ...pulseSnapshot, narratives: [] })).toContain('No narratives in this window.');
	});
});
