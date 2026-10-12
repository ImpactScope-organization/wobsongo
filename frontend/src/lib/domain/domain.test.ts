import { describe, expect, it } from 'vitest';
import { toCount } from './count';
import { isHighHarm, isSpiking } from './narrative';
import { parseTimeRange } from './time-range';
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
