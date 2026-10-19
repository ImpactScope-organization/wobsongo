import { describe, expect, it } from 'vitest';
import ActivityIcon from '@lucide/svelte/icons/activity';
import LayersIcon from '@lucide/svelte/icons/layers';
import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
import { activeNavHref, comingSoonSection, NAV_ITEMS, resolveNavIcon } from './navigation';

describe('NAV_ITEMS', () => {
	it('enables only Overview for now', () => {
		expect(NAV_ITEMS.filter((item) => !item.disabled).map((item) => item.href)).toEqual(['/']);
	});

	it('assigns an icon to each item', () => {
		for (const item of NAV_ITEMS) {
			expect(item.icon).toBeDefined();
		}
		expect(NAV_ITEMS[0].icon).toBe(LayoutDashboardIcon);
	});
});

describe('resolveNavIcon', () => {
	it('uses item.icon when present', () => {
		expect(resolveNavIcon(NAV_ITEMS[0])).toBe(LayoutDashboardIcon);
	});

	it('infers icon by path or label when icon is omitted', () => {
		expect(resolveNavIcon({ label: 'Ops', href: '/ops' })).toBe(ActivityIcon);
		expect(resolveNavIcon({ label: 'Unknown', href: '/custom' })).toBe(LayersIcon);
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
