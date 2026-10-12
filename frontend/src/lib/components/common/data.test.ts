import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { activity, PULSE_NOW } from '#lib/fixtures/pulse.js';
import { htmlSnippet, stripComments } from '#lib/testing/snippet.js';
import ActivityLog from './activity-log.svelte';
import KpiGroup from './kpi-group.svelte';
import KpiTile from './kpi-tile.svelte';
import ReviewActions from './review-actions.svelte';
import VerdictMeter from './verdict-meter.svelte';

describe('KpiGroup and KpiTile', () => {
	it('render as a description list', () => {
		const { body } = render(KpiGroup, { props: { children: htmlSnippet('<div></div>') } });
		expect(stripComments(body)).toMatch(/^<dl/);
	});

	it('pairs the label with its value', () => {
		const { body } = render(KpiTile, {
			props: { label: 'Active narratives', value: '37', caption: '6 new today' }
		});
		expect(body).toMatch(/<dt[^>]*>Active narratives<\/dt>/);
		expect(body).toContain('37');
		expect(body).toContain('6 new today');
	});

	it('supports a suffix, signal caption and visuals', () => {
		const { body } = render(KpiTile, {
			props: {
				label: 'Rated false or misleading',
				value: '21',
				suffix: '/ 37',
				caption: '▲ 64%',
				captionTone: 'signal',
				aside: htmlSnippet('<svg data-aside></svg>'),
				children: htmlSnippet('<div data-meter></div>')
			}
		});
		expect(body).toContain('/ 37');
		expect(body).toContain('data-tone="signal"');
		expect(body).toContain('data-aside');
		expect(body).toContain('data-meter');
	});
});

describe('VerdictMeter', () => {
	it('describes the breakdown in text', () => {
		const { body } = render(VerdictMeter, {
			props: { counts: { false: 14, misleading: 7, unverified: 9, accurate: 7 } }
		});
		expect(body).toContain('role="img"');
		expect(body).toContain('aria-label="14 false, 7 misleading, 9 unverified, 7 accurate"');
	});

	it('sizes segments by count and skips empty ones', () => {
		const { body } = render(VerdictMeter, {
			props: { counts: { false: 3, misleading: 0, unverified: 1, accurate: 0 } }
		});
		expect(body).toMatch(/data-segment="false"[^>]*flex-grow: 3/);
		expect(body).not.toContain('data-segment="misleading"');
	});

	it('handles nothing rated yet', () => {
		const { body } = render(VerdictMeter, {
			props: { counts: { false: 0, misleading: 0, unverified: 0, accurate: 0 } }
		});
		expect(body).toContain('aria-label="No rated narratives"');
	});
});

describe('ActivityLog', () => {
	it('lists entries with machine-readable times relative to now', () => {
		const { body } = render(ActivityLog, { props: { entries: activity, now: PULSE_NOW } });
		expect(body).toContain('datetime="2026-10-12T14:02:00.000Z"');
		expect(body).toContain('Today 14:02');
		expect(body).toContain('Yesterday 09:14');
		expect(body).toContain('Oct 9, 08:30');
	});

	it('names system events', () => {
		const { body } = render(ActivityLog, { props: { entries: activity, now: PULSE_NOW } });
		expect(body).toContain('System');
		expect(body).toContain('Dr. A.');
	});

	it('shows an empty state', () => {
		const { body } = render(ActivityLog, { props: { entries: [], now: PULSE_NOW } });
		expect(body).toContain('No activity yet.');
		expect(body).not.toContain('<ol');
	});
});

describe('ReviewActions', () => {
	const base = {
		itemId: 'c1',
		itemLabel: 'Breastfeeding always prevents pregnancy',
		action: '?/review'
	};

	it('is a plain POST form carrying the item id', () => {
		const { body } = render(ReviewActions, { props: { ...base, state: { kind: 'pending' } } });
		expect(body).toMatch(/<form[^>]*method="POST"[^>]*action="\?\/review"/);
		expect(body).toContain('name="id" value="c1"');
		expect(body).toContain('Review “Breastfeeding always prevents pregnancy”');
	});

	it('offers each pending decision as a submit button', () => {
		const { body } = render(ReviewActions, { props: { ...base, state: { kind: 'pending' } } });
		for (const decision of ['approve', 'override', 'dismiss']) {
			expect(body).toMatch(new RegExp(`type="submit" name="decision" value="${decision}"`));
		}
		expect(body).not.toContain('value="undo"');
	});

	it('lets callers choose the decisions', () => {
		const { body } = render(ReviewActions, {
			props: { ...base, state: { kind: 'pending' }, decisions: ['add-fact-check', 'dismiss'] }
		});
		expect(body).toContain('Add fact-check');
		expect(body).not.toContain('value="approve"');
	});

	it('shows the outcome and an undo button once decided', () => {
		const { body } = render(ReviewActions, {
			props: { ...base, state: { kind: 'decided', decision: 'override' } }
		});
		expect(body).toContain('Override saved · reply updated');
		expect(body).toContain('value="undo"');
		expect(body).not.toContain('value="approve"');
	});
});
