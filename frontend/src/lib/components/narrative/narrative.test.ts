import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { toCount } from '#lib/domain/count.js';
import {
	debunkFormats,
	engineVerdict,
	formats,
	languages,
	narrative,
	NARRATIVE_NOW,
	regions,
	reviewLog,
	samples,
	spread
} from '#lib/fixtures/narrative.js';
import { withLocale } from '#lib/testing/locale.js';
import { htmlSnippet } from '#lib/testing/snippet.js';
import AudienceBreakdown from './audience-breakdown.svelte';
import DebunkFormats from './debunk-formats.svelte';
import DismissMenu from './dismiss-menu.svelte';
import NarrativeHeader from './narrative-header.svelte';
import NarrativeMetrics from './narrative-metrics.svelte';
import RegionList from '../common/region-list.svelte';
import SampleMessages from './sample-messages.svelte';
import SpreadTimeline from './spread-timeline.svelte';
import VerdictPanel from './verdict-panel.svelte';

describe('NarrativeHeader', () => {
	it('shows the title as the h1 with its verdict, status, domain and summary', () => {
		const { body } = render(NarrativeHeader, { props: { narrative } });
		expect(body).toMatch(/<h1[^>]*>\s*Compulsory contraceptive trial for schoolgirls/);
		expect(body).toContain('data-verdict="false"');
		expect(body).toContain('data-status="needs-check"');
		expect(body).toContain('Family planning');
		expect(body).toContain('without parental consent');
	});

	it('renders page actions', () => {
		const { body } = render(NarrativeHeader, {
			props: { narrative, actions: htmlSnippet('<a href="/edit">Edit reply</a>') }
		});
		expect(body).toContain('href="/edit"');
	});
});

describe('DismissMenu', () => {
	it('is a <details> menu, so it opens without JavaScript', () => {
		const { body } = render(DismissMenu, { props: { narrativeId: 'N-0412', action: '?/dismiss' } });
		expect(body).toMatch(/<details[^>]*>\s*<summary/);
		expect(body).not.toMatch(/<details[^>]*\sopen/);
	});

	it('posts the narrative id and the chosen reason', () => {
		const { body } = render(DismissMenu, { props: { narrativeId: 'N-0412', action: '?/dismiss' } });
		expect(body).toMatch(/<form[^>]*method="POST"[^>]*action="\?\/dismiss"/);
		expect(body).toContain('name="id" value="N-0412"');
		for (const reason of ['not-srh', 'not-a-claim', 'duplicate', 'satire']) {
			expect(body).toContain(`type="submit" name="reason" value="${reason}"`);
		}
		expect(body).toContain('Already covered by another narrative');
	});

	it('can start open', () => {
		const { body } = render(DismissMenu, {
			props: { narrativeId: 'N-0412', action: '?/dismiss', open: true }
		});
		expect(body).toMatch(/<details[^>]*\sopen/);
	});
});

describe('NarrativeMetrics', () => {
	it('shows the four metrics', () => {
		const { body } = render(NarrativeMetrics, { props: { narrative } });
		expect(body).toContain('Messages, 7 d');
		expect(body).toContain('2,031');
		expect(body).toContain('×4.1');
		expect(body).toContain('Priority score');
		expect(body).toContain('87');
	});

	it('highlights fast growth and expert validation', () => {
		const { body } = render(NarrativeMetrics, { props: { narrative } });
		expect(body).toMatch(/data-tone="signal"[^>]*>\s*×4\.1/);
		expect(body).toContain('Expert-validated');
	});

	it('stays neutral for slow growth and unvalidated harm', () => {
		const { body } = render(NarrativeMetrics, {
			props: { narrative: { ...narrative, growthFactor6h: 1.2, harmValidated: false } }
		});
		expect(body).not.toMatch(/data-tone="signal"[^>]*>\s*×1\.2/);
		expect(body).not.toContain('Expert-validated');
	});
});

describe('SpreadTimeline', () => {
	it('draws one line per platform on the server', () => {
		const { body } = render(SpreadTimeline, { props: { spread, now: NARRATIVE_NOW } });
		expect(body.match(/<path[^>]*class="[^"]*lc-path/g)).toHaveLength(3);
		expect(body).toContain('viewBox="0 0 640 230"');
	});

	it('marks first sighting and the WhatsApp lag', () => {
		const { body } = render(SpreadTimeline, { props: { spread, now: NARRATIVE_NOW } });
		expect(body).toContain('First seen');
		expect(body).toContain('TikTok · Fri 16:15');
		expect(body).toContain('+18 h');
		expect(body).toContain('public → WhatsApp');
		expect(body).toContain('>Now<');
	});

	it('describes the story for screen readers', () => {
		const { body } = render(SpreadTimeline, { props: { spread, now: NARRATIVE_NOW } });
		expect(body).toContain(
			'aria-label="Mentions per hour by platform: TikTok, Facebook, WhatsApp. First seen on TikTok, Fri 16:15. Reached WhatsApp Sat 10:30, 18 h later."'
		);
	});

	it('handles a narrative that has not reached WhatsApp', () => {
		const { body } = render(SpreadTimeline, {
			props: { spread: { ...spread, firstOnWhatsApp: null }, now: NARRATIVE_NOW }
		});
		expect(body).toContain('Not on WhatsApp yet.');
		expect(body).not.toContain('public → WhatsApp');
	});

	const circles = (body: string) =>
		[...body.matchAll(/<circle cx="([-\d.]+)" cy="([-\d.]+)"/g)].map(([, cx, cy]) => ({
			cx: Number(cx),
			cy: Number(cy)
		}));

	it('keeps a first sighting from before the series on the chart', () => {
		const firstPoint = spread.series[0].points[0].at.getTime();
		const early = { ...spread.firstSeen, at: new Date(firstPoint - 6 * 3_600_000) };
		const { body } = render(SpreadTimeline, {
			props: { spread: { ...spread, firstSeen: early }, now: NARRATIVE_NOW }
		});
		for (const { cx } of circles(body)) expect(cx).toBeGreaterThanOrEqual(0);
	});

	it('draws all-zero series on the baseline', () => {
		const flat = spread.series.map((line) => ({
			...line,
			points: line.points.map((point) => ({ ...point, count: 0 }))
		}));
		const { body } = render(SpreadTimeline, {
			props: { spread: { ...spread, series: flat }, now: NARRATIVE_NOW }
		});
		expect(body).not.toContain('NaN');
		const baseline = circles(body)[0].cy;
		const ys = [...body.matchAll(/<path d="M[-\d.]+,([-\d.]+)/g)].map(([, y]) => Number(y));
		expect(ys.length).toBeGreaterThan(0);
		for (const y of ys) expect(y).toBeCloseTo(baseline);
	});

	it('moves the WhatsApp labels down a line when they would overlap the first-seen ones', () => {
		const soon = new Date(spread.firstSeen.at.getTime() + 3 * 3_600_000);
		const { body } = render(SpreadTimeline, {
			props: { spread: { ...spread, firstOnWhatsApp: soon }, now: NARRATIVE_NOW }
		});
		expect(body).toMatch(/y="42"[^>]*>WhatsApp</);
		expect(render(SpreadTimeline, { props: { spread, now: NARRATIVE_NOW } }).body).toMatch(
			/y="14"[^>]*>WhatsApp</
		);
	});
});

describe('VerdictPanel', () => {
	const props = {
		narrativeId: narrative.id,
		narrativeTitle: narrative.title,
		engine: engineVerdict,
		state: { kind: 'pending' } as const,
		action: '?/review',
		overrideHref: '/narratives/N-0412/reply',
		log: reviewLog,
		now: NARRATIVE_NOW
	};

	it('shows the engine verdict, confidence and explanation', () => {
		const { body } = render(VerdictPanel, { props });
		expect(body).toMatch(/False · 0\.91<span class="sr-only">(&nbsp;|\u00a0)confidence/);
		expect(body).toContain('Matched to “No compulsory contraception trials');
	});

	it('offers confirm and correct as review decisions', () => {
		const { body } = render(VerdictPanel, { props });
		expect(body).toMatch(/value="approve"[^>]*>\s*Confirm verdict/);
		expect(body).toMatch(/<a href="\/narratives\/N-0412\/reply"[^>]*>\s*Correct…/);
		expect(body).not.toContain('value="dismiss"');
	});

	it('lists the review log with clock times', () => {
		const { body } = render(VerdictPanel, { props });
		expect(body).toContain('>13:52<');
		expect(body).toContain('Narrative created: 3 clusters merged');
	});
});

describe('AudienceBreakdown', () => {
	it('lists languages with shares', () => {
		const { body } = render(AudienceBreakdown, { props: { languages, formats } });
		expect(body).toContain('Messages by language');
		expect(body).toMatch(/Mooré<\/th>[\s\S]*?58%/);
	});

	it('shows the format mix as a labelled stacked bar', () => {
		const { body } = render(AudienceBreakdown, { props: { languages, formats } });
		expect(body).toContain('aria-label="Voice note 52%, Video 23%, Text 17%, Image 8%"');
	});
});

describe('RegionList', () => {
	it('shows counts and suppresses small ones', () => {
		const { body } = render(RegionList, { props: { regions } });
		expect(body).toContain('Centre');
		expect(body).toContain('412');
		expect(body.match(/data-suppressed="true"/g)).toHaveLength(2);
		expect(body).not.toMatch(/>\s*6\s*</);
	});

	it('renders an optional map', () => {
		const { body } = render(RegionList, {
			props: {
				regions: [{ region: 'Centre', count: toCount(20) }],
				map: htmlSnippet('<svg data-map></svg>')
			}
		});
		expect(body).toContain('data-map');
	});

	it('shows an empty state', () => {
		expect(render(RegionList, { props: { regions: [] } }).body).toContain(
			'No location answers yet.'
		);
	});
});

describe('SampleMessages', () => {
	it('quotes each anonymised message with its provenance', () => {
		const { body } = render(SampleMessages, { props: { samples } });
		expect(body.match(/<blockquote/g)).toHaveLength(3);
		expect(body).toContain('Voice note 0:42');
		expect(body).toContain('Auto-transcribed, translated from Mooré');
	});

	it('shows an empty state', () => {
		expect(render(SampleMessages, { props: { samples: [] } }).body).toContain(
			'No sample messages.'
		);
	});
});

describe('DebunkFormats', () => {
	it('lists formats with reach and availability', () => {
		const { body } = render(DebunkFormats, { props: { formats: debunkFormats } });
		expect(body).toContain('Audio · Mooré');
		expect(body).toContain('Text · French');
		expect(body).toContain('58% of messages');
		expect(body.match(/data-available="false"/g)).toHaveLength(2);
		expect(body).toContain('Missing');
	});
});

describe('in French', () => {
	it('words the dismiss menu, metrics and timeline in French', () => {
		const [menu, metrics, timeline] = withLocale('fr', () => [
			render(DismissMenu, { props: { narrativeId: 'N-0412', action: '?/dismiss' } }).body,
			render(NarrativeMetrics, { props: { narrative } }).body,
			render(SpreadTimeline, { props: { spread, now: NARRATIVE_NOW } }).body
		]);
		expect(menu).toContain('Écarter le récit comme');
		expect(menu).toContain('Déjà couvert par un autre récit');
		expect(metrics).toContain('Score de préjudice');
		expect(metrics).toContain('Messages, 7 j');
		expect(timeline).toContain('Mentions par heure et par plateforme\u00a0:');
		expect(timeline).toContain('Maintenant');
	});
});
