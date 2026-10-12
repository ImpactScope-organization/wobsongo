import { describe, expect, it } from 'vitest';
import {
	formatClockTime,
	formatLogTime,
	formatNumber,
	formatPercent,
	percentShares
} from './format';

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
