import { describe, expect, it } from 'vitest';
import {
	formatAgo,
	formatClockTime,
	formatCountdown,
	formatDate,
	formatDayTime,
	formatFactor,
	formatIsoWeek,
	formatLogTime,
	formatNumber,
	formatPercent,
	formatSince,
	formatTotal,
	percentShares,
	formatWeekday,
	plural
} from './format';
import { withLocale } from '#lib/testing/locale.js';

describe('formatNumber', () => {
	it('groups thousands', () => {
		expect(formatNumber(8412)).toBe('8,412');
		expect(formatNumber(0)).toBe('0');
	});
});

describe('formatPercent', () => {
	it('rounds to whole percent', () => {
		expect(formatPercent(46.4)).toBe('46%');
	});

	it('signs positive and negative values when asked', () => {
		expect(formatPercent(412, { signed: true })).toBe('+412%');
		expect(formatPercent(-3, { signed: true })).toBe('−3%');
	});

	it('never signs zero, including values that round to zero', () => {
		expect(formatPercent(0, { signed: true })).toBe('0%');
		expect(formatPercent(-0.2, { signed: true })).toBe('0%');
		expect(formatPercent(-0.2)).toBe('0%');
	});
});

describe('percentShares', () => {
	it('sums to exactly 100', () => {
		const shares = percentShares([2693, 1571, 1099, 272, 153]);
		expect(shares).toEqual([46, 27, 19, 5, 3]);
		expect(shares.reduce((a, b) => a + b, 0)).toBe(100);
	});

	it('gives the leftover points to the largest remainders', () => {
		expect(percentShares([1, 1, 1])).toEqual([34, 33, 33]);
	});

	it('returns zeros when there is nothing to share', () => {
		expect(percentShares([0, 0])).toEqual([0, 0]);
		expect(percentShares([])).toEqual([]);
	});
});

describe('formatClockTime', () => {
	it('uses a 24-hour clock in UTC by default', () => {
		expect(formatClockTime(new Date('2026-10-12T14:05:00Z'))).toBe('14:05');
		expect(formatClockTime(new Date('2026-10-12T00:30:00Z'))).toBe('00:30');
	});
});

describe('formatLogTime', () => {
	const now = new Date('2026-10-12T14:05:00Z');

	it('says Today for the same day', () => {
		expect(formatLogTime(new Date('2026-10-12T08:30:00Z'), now)).toBe('Today 08:30');
	});

	it('says Yesterday for the previous day', () => {
		expect(formatLogTime(new Date('2026-10-11T23:59:00Z'), now)).toBe('Yesterday 23:59');
	});

	it('shows the date for anything older', () => {
		expect(formatLogTime(new Date('2026-10-09T14:02:00Z'), now)).toBe('Oct 9, 14:02');
	});
});

describe('formatDayTime', () => {
	it('shows the short weekday and 24-hour time in UTC', () => {
		expect(formatDayTime(new Date('2026-10-12T02:10:00Z'))).toBe('Mon 02:10');
	});
});

describe('formatFactor', () => {
	it('shows one decimal with a multiplication sign', () => {
		expect(formatFactor(4.1)).toBe('×4.1');
		expect(formatFactor(1)).toBe('×1.0');
	});
});

describe('formatAgo', () => {
	const now = new Date('2026-10-12T14:05:00Z');
	const ago = (ms: number) => formatAgo(new Date(now.getTime() - ms), now);

	it('coarsens elapsed time to minutes, hours or days', () => {
		expect(ago(30_000)).toBe('just now');
		expect(ago(38 * 60_000)).toBe('38 min ago');
		expect(ago(2 * 3_600_000 + 59 * 60_000)).toBe('2 h ago');
		expect(ago(3 * 86_400_000)).toBe('3 d ago');
	});

	it('treats future times as just now', () => {
		expect(ago(-60_000)).toBe('just now');
	});
});

describe('formatWeekday', () => {
	it('shows the short weekday in UTC', () => {
		expect(formatWeekday(new Date('2026-10-12T23:30:00Z'))).toBe('Mon');
	});
});

describe('formatIsoWeek', () => {
	it('numbers weeks from the one holding the first Thursday', () => {
		expect(formatIsoWeek(new Date('2026-08-17T00:00:00Z'))).toBe('W34');
		expect(formatIsoWeek(new Date('2026-10-11T23:59:00Z'))).toBe('W41');
		expect(formatIsoWeek(new Date('2026-10-12T00:00:00Z'))).toBe('W42');
	});

	it('handles weeks that cross a year boundary', () => {
		expect(formatIsoWeek(new Date('2025-12-29T00:00:00Z'))).toBe('W1');
		expect(formatIsoWeek(new Date('2027-01-01T00:00:00Z'))).toBe('W53');
	});
});

describe('formatTotal', () => {
	it('marks rounded totals as approximate', () => {
		expect(formatTotal({ value: 902, approximate: false })).toBe('n = 902');
		expect(formatTotal({ value: 6220, approximate: true })).toBe('n ≈ 6,220');
	});
});

describe('formatSince', () => {
	const now = new Date('2026-10-12T14:05:00Z');
	const ago = (days: number) => new Date(now.getTime() - days * 86_400_000);

	it('coarsens long spans', () => {
		expect(formatSince(ago(0.5), now)).toBe('today');
		expect(formatSince(ago(1), now)).toBe('yesterday');
		expect(formatSince(ago(5), now)).toBe('5 days ago');
		expect(formatSince(ago(14), now)).toBe('2 weeks ago');
		expect(formatSince(ago(96), now)).toBe('3 months ago');
		expect(formatSince(ago(400), now)).toBe('1 year ago');
	});
});

describe('formatDate', () => {
	it('reads day month year', () => {
		expect(formatDate(new Date('2026-09-02T10:12:00Z'))).toBe('2 Sep 2026');
	});
});

describe('formatCountdown', () => {
	it('shows minutes and seconds, rounding up and never negative', () => {
		expect(formatCountdown(552_000)).toBe('9:12');
		expect(formatCountdown(59_001)).toBe('1:00');
		expect(formatCountdown(-5)).toBe('0:00');
	});
});

describe('in French', () => {
	const now = new Date('2026-10-12T14:05:00Z');
	const fr = <T>(run: () => T) => withLocale('fr', run);

	it('groups thousands with a narrow no-break space', () => {
		expect(fr(() => formatNumber(8412))).toBe('8\u202f412');
	});

	it('words dates and relative times in French', () => {
		expect(fr(() => formatDate(new Date('2026-09-02T10:00:00Z')))).toBe('2 sept. 2026');
		expect(fr(() => formatLogTime(new Date('2026-10-12T09:14:00Z'), now))).toBe(
			'Aujourd’hui 09:14'
		);
		expect(fr(() => formatAgo(new Date('2026-10-12T13:27:00Z'), now))).toBe('il y a 38 min');
		expect(fr(() => formatSince(new Date('2026-10-11T14:05:00Z'), now))).toBe('hier');
		expect(fr(() => formatSince(new Date('2026-10-07T14:05:00Z'), now))).toBe('il y a 5 jours');
		expect(fr(() => formatIsoWeek(new Date('2026-08-20T00:00:00Z')))).toBe('S34');
	});

	it('counts 0 as singular, unlike English', () => {
		expect(plural(0, 'one', 'other')).toBe('other');
		expect(fr(() => plural(0, 'one', 'other'))).toBe('one');
		expect(fr(() => plural(2, 'one', 'other'))).toBe('other');
	});
});
