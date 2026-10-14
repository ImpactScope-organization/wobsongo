import { describe, expect, it } from 'vitest';
import { botLinkMessage, initials } from './account';
import { toCount, toTotal } from './count';
import { isHighHarm, isSpiking } from './narrative';
import { parseReviewFormDecision } from './review';
import { parseTimeRange } from './time-range';
import { whatsAppLagHours } from './narrative-detail';
import { fixedClock } from '../utils/clock';

describe('toCount', () => {
	it('suppresses counts below the minimum cell size', () => {
		expect(toCount(6)).toEqual({ kind: 'suppressed', threshold: 10 });
		expect(toCount(4, 5)).toEqual({ kind: 'suppressed', threshold: 5 });
	});

	it('keeps counts at or above the minimum cell size', () => {
		expect(toCount(10)).toEqual({ kind: 'value', value: 10 });
		expect(toCount(412)).toEqual({ kind: 'value', value: 412 });
	});
});

describe('isHighHarm', () => {
	it('is true from harm 4 upwards', () => {
		expect(isHighHarm(3)).toBe(false);
		expect(isHighHarm(4)).toBe(true);
		expect(isHighHarm(5)).toBe(true);
	});
});

describe('isSpiking', () => {
	it('is true only above 50% growth', () => {
		expect(isSpiking(50)).toBe(false);
		expect(isSpiking(51)).toBe(true);
		expect(isSpiking(-80)).toBe(false);
	});
});

describe('parseTimeRange', () => {
	it('accepts known ranges', () => {
		expect(parseTimeRange('7d')).toBe('7d');
	});

	it('falls back to 24h for missing or unknown values', () => {
		expect(parseTimeRange(null)).toBe('24h');
		expect(parseTimeRange('1y')).toBe('24h');
	});
});

describe('fixedClock', () => {
	it('always returns the same instant, as a fresh Date', () => {
		const clock = fixedClock('2026-10-12T14:05:00Z');
		const first = clock.now();
		first.setFullYear(2000);
		expect(clock.now().toISOString()).toBe('2026-10-12T14:05:00.000Z');
	});
});

describe('whatsAppLagHours', () => {
	const firstSeen = { platform: 'tiktok' as const, at: new Date('2026-10-12T02:10:00Z') };

	it('is the rounded hours from first sighting to first WhatsApp submission', () => {
		expect(
			whatsAppLagHours({
				series: [],
				firstSeen,
				firstOnWhatsApp: new Date('2026-10-12T20:25:00Z')
			})
		).toBe(18);
	});

	it('is null before the narrative reaches WhatsApp', () => {
		expect(whatsAppLagHours({ series: [], firstSeen, firstOnWhatsApp: null })).toBeNull();
	});

	it('is never negative when WhatsApp saw it first', () => {
		expect(
			whatsAppLagHours({
				series: [],
				firstSeen,
				firstOnWhatsApp: new Date('2026-10-12T01:00:00Z')
			})
		).toBe(0);
	});
});

describe('parseReviewFormDecision', () => {
	it('accepts what a review form posts, including undo', () => {
		for (const value of ['approve', 'dismiss', 'add-fact-check', 'undo']) {
			expect(parseReviewFormDecision(value)).toBe(value);
		}
	});

	it('rejects override, which goes through the editor page, and anything else', () => {
		expect(parseReviewFormDecision('override')).toBeNull();
		expect(parseReviewFormDecision(null)).toBeNull();
		expect(parseReviewFormDecision('APPROVE')).toBeNull();
	});
});

describe('toTotal', () => {
	it('keeps the exact total when nothing is suppressed', () => {
		expect(toTotal(6219, [toCount(3120), toCount(42)])).toEqual({
			value: 6219,
			approximate: false
		});
	});

	it('rounds the total when a part is suppressed, so it cannot be subtracted back', () => {
		expect(toTotal(6219, [toCount(6212), toCount(7)])).toEqual({ value: 6220, approximate: true });
	});
});

describe('initials', () => {
	it('takes the first and last name, skipping titles', () => {
		expect(initials('Dr. Aminata Ouédraogo')).toBe('AO');
		expect(initials('Kader Sawadogo Zongo')).toBe('KZ');
		expect(initials('awa')).toBe('A');
		expect(initials('  ')).toBe('?');
	});
});

describe('botLinkMessage', () => {
	it('prefixes the code so the bot never fact-checks it', () => {
		expect(botLinkMessage({ code: '482913' })).toBe('WOBSONGO-AUTH 482-913');
	});
});
