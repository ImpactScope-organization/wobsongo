import { describe, expect, it } from 'vitest';
import { withSearchParam } from './url';

describe('withSearchParam', () => {
	it('sets one param and keeps the others', () => {
		const url = new URL('https://example.org/?range=7d&topic=all');
		expect(withSearchParam(url, 'range', '30d')).toBe('?range=30d&topic=all');
		expect(withSearchParam(url, 'topic', 'misinfo')).toBe('?range=7d&topic=misinfo');
	});

	it('adds the param when absent', () => {
		expect(withSearchParam(new URL('https://example.org/'), 'range', '24h')).toBe('?range=24h');
	});
});
