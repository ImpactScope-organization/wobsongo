import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { REVIEW_STATUSES, VERDICTS } from '#lib/domain/narrative.js';
import { toCount } from '#lib/domain/count.js';
import { withLocale } from '#lib/testing/locale.js';
import CountValue from './count-value.svelte';
import HarmScore from './harm-score.svelte';
import ReviewStatusBadge from './review-status-badge.svelte';
import VerdictBadge from './verdict-badge.svelte';

describe('VerdictBadge', () => {
	it.each(VERDICTS)('labels the %s verdict for sighted and screen-reader users', (verdict) => {
		const { body } = render(VerdictBadge, { props: { verdict } });
		expect(body).toContain(`data-verdict="${verdict}"`);
		expect(body).toContain('Verdict:');
	});

	it('uses the human label, not the code', () => {
		expect(render(VerdictBadge, { props: { verdict: 'misleading' } }).body).toContain('Misleading');
	});
});

describe('ReviewStatusBadge', () => {
	it.each(REVIEW_STATUSES)('renders the %s status', (status) => {
		const { body } = render(ReviewStatusBadge, { props: { status } });
		expect(body).toContain(`data-status="${status}"`);
	});

	it('spells out the knowledge-base status', () => {
		expect(render(ReviewStatusBadge, { props: { status: 'not-in-kb' } }).body).toContain(
			'Not in KB'
		);
	});
});

describe('HarmScore', () => {
	it('shows the score out of five', () => {
		expect(render(HarmScore, { props: { harm: 3 } }).body).toContain('Harm 3/5');
	});

	it('flags high harm', () => {
		expect(render(HarmScore, { props: { harm: 4 } }).body).toContain('data-high-harm="true"');
		expect(render(HarmScore, { props: { harm: 3 } }).body).not.toContain('data-high-harm=');
	});

	it('has a large variant for metric tiles', () => {
		const { body } = render(HarmScore, { props: { harm: 5, size: 'large' } });
		expect(body).toContain('text-[2rem]');
		expect(body).toMatch(/5<span[^>]*>\/5<\/span>/);
	});
});

describe('CountValue', () => {
	it('formats visible counts', () => {
		expect(render(CountValue, { props: { count: toCount(3120) } }).body).toContain('3,120');
	});

	it('hides suppressed counts and says why', () => {
		const { body } = render(CountValue, { props: { count: toCount(6) } });
		expect(body).not.toContain('>6<');
		expect(body).toContain('&lt; 10');
		expect(body).toContain('hidden to protect privacy');
	});
});

describe('badges in French', () => {
	it('words verdicts, harm and suppressed counts', () => {
		const verdict = withLocale(
			'fr',
			() => render(VerdictBadge, { props: { verdict: 'misleading' } }).body
		);
		expect(verdict).toContain('Trompeur');
		const harm = withLocale('fr', () => render(HarmScore, { props: { harm: 3 } }).body);
		expect(harm).toContain('Nocivité 3/5');
		const count = withLocale(
			'fr',
			() => render(CountValue, { props: { count: { kind: 'suppressed', threshold: 5 } } }).body
		);
		expect(count).toContain('Moins de 5\u00a0: masqué pour protéger la vie privée');
	});
});
