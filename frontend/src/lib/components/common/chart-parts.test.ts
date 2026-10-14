import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { toCount } from '#lib/domain/count.js';
import { htmlSnippet, stripComments } from '#lib/testing/snippet.js';
import BarList from './bar-list.svelte';
import InsightCallout from './insight-callout.svelte';
import RegionList from './region-list.svelte';
import ShareColumns from './share-columns.svelte';
import SwatchLegend from './swatch-legend.svelte';

const column = (key: string, a: number, b: number) => ({
	key,
	label: key,
	segments: [
		{ key: 'a', label: 'Audio', value: a, class: 'bg-primary' },
		{ key: 'b', label: 'Text', value: b, class: 'bg-rule' }
	]
});

describe('ShareColumns', () => {
	it('hides the visual and carries the numbers in a table', () => {
		const body = stripComments(
			render(ShareColumns, {
				props: { caption: 'Format by week', columns: [column('W1', 3, 1), column('W2', 1, 1)] }
			}).body
		);
		expect(body).toMatch(/<div aria-hidden="true"[^>]*>/);
		expect(body).toContain('<caption>Format by week</caption>');
		expect(body).toMatch(
			/<th scope="col">Week<\/th><th scope="col">Audio<\/th><th scope="col">Text<\/th>/
		);
		expect(body).toMatch(/<th scope="row">W1<\/th><td>75%<\/td><td>25%<\/td>/);
		expect(body).toMatch(/<th scope="row">W2<\/th><td>50%<\/td><td>50%<\/td>/);
	});

	it('sizes segments by value and skips empty ones', () => {
		const { body } = render(ShareColumns, {
			props: { caption: 'x', columns: [column('W1', 3, 0)] }
		});
		expect(body).toContain('data-segment="a"');
		expect(body).toContain('flex-grow: 3');
		expect(body).not.toContain('data-segment="b"');
	});

	it('lines table cells up by segment key when a column lacks one', () => {
		const partial = { key: 'W2', label: 'W2', segments: [column('W2', 0, 2).segments[1]] };
		const body = stripComments(
			render(ShareColumns, { props: { caption: 'x', columns: [column('W1', 1, 1), partial] } }).body
		);
		expect(body).toMatch(/<th scope="row">W2<\/th><td>—<\/td><td>100%<\/td>/);
	});

	it('shows an empty state', () => {
		const { body } = render(ShareColumns, { props: { caption: 'x', columns: [] } });
		expect(body).toContain('No data in this window.');
	});
});

describe('SwatchLegend', () => {
	const items = [{ key: 'a', label: 'Audio', class: 'bg-primary' }];

	it('is hidden from screen readers by default', () => {
		// Each swatch is hidden anyway, so check the list itself.
		expect(render(SwatchLegend, { props: { items } }).body).toMatch(/<ul aria-hidden="true"/);
	});

	it('is read out when it carries descriptions', () => {
		const { body } = render(SwatchLegend, {
			props: {
				decorative: false,
				items: [{ ...items[0], description: 'Voice notes and audio files' }]
			}
		});
		expect(body).not.toMatch(/<ul aria-hidden/);
		expect(body).toContain('Voice notes and audio files');
	});
});

describe('InsightCallout', () => {
	it('leads with the finding in bold', () => {
		const { body } = render(InsightCallout, {
			props: { title: 'Voice notes now 52%.', children: htmlSnippet('<span>Prefer audio.</span>') }
		});
		expect(body).toMatch(/<strong[^>]*>Voice notes now 52%.<\/strong>/);
		expect(body).toContain('Prefer audio.');
	});
});

describe('RegionList topics', () => {
	it('adds a topic line once any region has one, without revealing suppressed ones', () => {
		const body = stripComments(
			render(RegionList, {
				props: {
					regions: [
						{ region: 'Centre', count: toCount(3120), topTopic: 'Contraceptive trial rumor' },
						{ region: 'Est', count: toCount(42) },
						{ region: 'Sahel', count: toCount(7) }
					]
				}
			}).body
		);
		expect(body).toContain('Contraceptive trial rumor');
		expect(body).toMatch(/Est\s*<span[^>]*>—<\/span>/);
		expect(body).toContain('Too few responses to show');
	});

	it('has no topic lines when no region has a topic', () => {
		const { body } = render(RegionList, {
			props: { regions: [{ region: 'Centre', count: toCount(412) }] }
		});
		expect(body).not.toContain('Too few responses');
		expect(body).not.toContain('—');
	});
});

describe('BarList', () => {
	it('shows an empty state instead of an empty table', () => {
		const { body } = render(BarList, { props: { caption: 'Messages by language', items: [] } });
		expect(body).toContain('No data in this window.');
		expect(body).not.toContain('<table');
	});
});
