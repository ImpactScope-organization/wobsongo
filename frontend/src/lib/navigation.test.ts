import { describe, expect, it } from 'vitest';
import { activeNavHref, comingSoonSection, NAV_ITEMS } from './navigation';

describe('NAV_ITEMS', () => {
	it('enables only Overview for now', () => {
		expect(NAV_ITEMS.filter((item) => !item.disabled).map((item) => item.href)).toEqual(['/']);
	});
});

describe('activeNavHref', () => {
	it('matches Overview on the home page only', () => {
		expect(activeNavHref('/')).toBe('/');
		expect(activeNavHref('/somewhere')).toBeUndefined();
	});

	it('matches settings and the pages under it', () => {
		expect(activeNavHref('/settings')).toBe('/settings');
		expect(activeNavHref('/settings/bots/whatsapp')).toBe('/settings');
		expect(activeNavHref('/settingsx')).toBeUndefined();
	});

	it('prefers the longest match', () => {
		expect(activeNavHref('/a/b/c', ['/a', '/a/b'])).toBe('/a/b');
	});
});

describe('comingSoonSection', () => {
	it('finds the unbuilt view a path belongs to', () => {
		expect(comingSoonSection('/narratives/N-0412')?.label).toBe('Narratives');
		expect(comingSoonSection('/safety')?.label).toBe('High-harm');
	});

	it('is undefined for built or unknown paths', () => {
		expect(comingSoonSection('/')).toBeUndefined();
		expect(comingSoonSection('/settings')).toBeUndefined();
		expect(comingSoonSection('/spreadsheet')).toBeUndefined();
	});
});
