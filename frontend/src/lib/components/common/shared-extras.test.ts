import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { activity, PULSE_NOW } from '#lib/fixtures/pulse.js';
import { stripComments } from '#lib/testing/snippet.js';
import ActivityLog from './activity-log.svelte';
import BarList from './bar-list.svelte';
import KpiTile from './kpi-tile.svelte';
import PageBreadcrumb from './page-breadcrumb.svelte';
import ReviewActions from './review-actions.svelte';
import StackedBar from './stacked-bar.svelte';

const segments = [
	{ key: 'a', label: 'Voice note', value: 52, class: 'bg-primary' },
	{ key: 'b', label: 'Video', value: 23, class: 'bg-muted-foreground' },
	{ key: 'c', label: 'Image', value: 0, class: 'bg-secondary' },
	{ key: 'd', label: 'Text', value: 25, class: 'bg-rule' }
];

describe('StackedBar', () => {
	it('describes segments by count by default and skips empty ones', () => {
		const { body } = render(StackedBar, { props: { segments } });
		expect(body).toContain('aria-label="52 Voice note, 23 Video, 25 Text"');
		expect(body).not.toContain('data-segment="c"');
	});

	it('formats counts in the description', () => {
		const { body } = render(StackedBar, {
			props: { segments: [{ key: 'f', label: 'false', value: 2693, class: 'bg-primary' }] }
		});
		expect(body).toContain('aria-label="2,693 false"');
	});

	it('has a compact size that still prints percentages', () => {
		const { body } = render(StackedBar, {
			props: { segments, describe: 'percent', showPercent: true, size: 'md' }
		});
		expect(body).toContain('data-size="md"');
		expect(body).toContain('>52%</span>');
	});

	it('can describe by percent and print percentages in the large size', () => {
		const { body } = render(StackedBar, {
			props: { segments, describe: 'percent', showPercent: true, size: 'lg' }
		});
		expect(body).toContain('aria-label="Voice note 52%, Video 23%, Text 25%"');
		expect(body).toContain('>52%</span>');
	});

	it('does not print percentages in the thin size', () => {
		const { body } = render(StackedBar, { props: { segments, showPercent: true } });
		expect(body).not.toContain('>52%</span>');
	});

	it('uses the empty text when every value is zero', () => {
		const { body } = render(StackedBar, {
			props: { segments: [{ ...segments[2] }], emptyText: 'Nothing yet' }
		});
		expect(body).toContain('aria-label="Nothing yet"');
	});
});

describe('BarList', () => {
	const items = [
		{ key: 'm', label: 'Mooré', value: 75 },
		{ key: 'f', label: 'French', value: 25 }
	];

	it('is a captioned table with row headers', () => {
		const { body } = render(BarList, { props: { items, caption: 'Messages by language' } });
		expect(body).toContain('<caption class="sr-only">Messages by language</caption>');
		expect(body).toMatch(/<th scope="row"[^>]*>\s*Mooré\s*<\/th>/);
	});

	it('scales bars to the largest value and shows shares', () => {
		const { body } = render(BarList, { props: { items, caption: 'x' } });
		expect(body).toContain('width: 100%');
		expect(body).toMatch(/width: 33\.3+%/);
		expect(body).toContain('75%');
	});

	it('can show raw values instead', () => {
		const { body } = render(BarList, {
			props: { items: [{ key: 'p', label: 'Potash', value: 2100 }], caption: 'x', show: 'value' }
		});
		expect(body).toContain('2,100');
	});
});

describe('PageBreadcrumb', () => {
	it('links ancestors and marks the last crumb as the current page', () => {
		const { body } = render(PageBreadcrumb, {
			props: {
				crumbs: [
					{ label: 'Pulse', href: '/pulse' },
					{ label: 'Narratives' },
					{ label: 'N-0412', mono: true }
				]
			}
		});
		expect(body).toContain('aria-label="breadcrumb"');
		expect(body).toContain('href="/pulse"');
		expect(stripComments(body)).toMatch(/aria-current="page"[^>]*>\s*N-0412/);
		expect(body).not.toContain('href="undefined"');
	});
});

describe('ActivityLog compact variant', () => {
	it('shows clock times and hides the actor column', () => {
		const { body } = render(ActivityLog, {
			props: { entries: activity, now: PULSE_NOW, variant: 'compact' }
		});
		expect(body).toContain('>14:02<');
		expect(body).not.toContain('Today 14:02<');
		expect(body).toContain('title="Today 14:02"');
		expect(body).not.toContain('>System<');
	});
});

describe('ReviewActions labels', () => {
	it('lets callers rename decisions without changing the posted value', () => {
		const { body } = render(ReviewActions, {
			props: {
				itemId: 'N-0412',
				itemLabel: 'x',
				action: '?/review',
				state: { kind: 'pending' },
				decisions: ['approve', 'override'],
				overrideHref: '/narratives/N-0412/reply',
				labels: { approve: 'Confirm verdict', override: 'Correct…' }
			}
		});
		expect(body).toMatch(/value="approve"[^>]*>\s*Confirm verdict/);
		expect(body).toMatch(/href="\/narratives\/N-0412\/reply"[^>]*>\s*Correct…/);
	});
});

describe('KpiTile small size', () => {
	it('marks the size and can colour the value', () => {
		const { body } = render(KpiTile, {
			props: { label: 'Growth, last 6 h', value: '×4.1', size: 'sm', valueTone: 'signal' }
		});
		expect(body).toContain('data-size="sm"');
		expect(body).toMatch(/data-tone="signal"[^>]*>\s*×4\.1/);
	});
});
