import { describe, expect, it } from 'vitest';
import { isWhitelistedForFullData } from './privacy.js';

describe('isWhitelistedForFullData', () => {
	it('returns false when email or whitelist is not provided', () => {
		expect(isWhitelistedForFullData(undefined, 'a@example.com')).toBe(false);
		expect(isWhitelistedForFullData(null, 'a@example.com')).toBe(false);
		expect(isWhitelistedForFullData('', 'a@example.com')).toBe(false);
		expect(isWhitelistedForFullData('a@example.com', undefined)).toBe(false);
		expect(isWhitelistedForFullData('a@example.com', null)).toBe(false);
		expect(isWhitelistedForFullData('a@example.com', '')).toBe(false);
	});

	it('returns true when email matches single entry in whitelist', () => {
		expect(isWhitelistedForFullData('a@example.com', 'a@example.com')).toBe(true);
	});

	it('handles comma-separated lists and trims spaces', () => {
		const list = ' first@example.com , second@example.com,third@example.com ';
		expect(isWhitelistedForFullData('second@example.com', list)).toBe(true);
		expect(isWhitelistedForFullData('first@example.com', list)).toBe(true);
		expect(isWhitelistedForFullData('third@example.com', list)).toBe(true);
		expect(isWhitelistedForFullData('fourth@example.com', list)).toBe(false);
	});

	it('matches case-insensitively', () => {
		const list = 'Admin@WobSongo.ORG, LEAD@domain.com';
		expect(isWhitelistedForFullData('admin@wobsongo.org', list)).toBe(true);
		expect(isWhitelistedForFullData('ADMIN@WOBSONGO.ORG', list)).toBe(true);
		expect(isWhitelistedForFullData('lead@domain.com', list)).toBe(true);
		expect(isWhitelistedForFullData('other@domain.com', list)).toBe(false);
	});
});
