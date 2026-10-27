import { describe, expect, it } from 'vitest';
import ActivityIcon from '@lucide/svelte/icons/activity';
import LayersIcon from '@lucide/svelte/icons/layers';
import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
import ScanSearchIcon from '@lucide/svelte/icons/scan-search';
import { withLocale } from '#lib/testing/locale.js';
import { activeNavHref, comingSoonSection, navItems, resolveNavIcon } from './navigation';

describe('navItems', () => {
	it('enables Overview and Monitor for now', () => {
		expect(
			navItems()
				.filter((item) => !item.disabled)
				.map((item) => item.href)
		).toEqual(['/', '/monitor']);
	});

	it('assigns an icon to each item', () => {
		for (const item of navItems()) {
			expect(item.icon).toBeDefined();
		}
		expect(navItems()[0].icon).toBe(LayoutDashboardIcon);
	});

	it('labels the items in the current language', () => {
		expect(navItems()[0].label).toBe('Overview');
		expect(withLocale('fr', () => navItems()[0].label)).toBe('Vue d’ensemble');
	});
});

describe('resolveNavIcon', () => {
	it('uses item.icon when present', () => {
		expect(resolveNavIcon(navItems()[0])).toBe(LayoutDashboardIcon);
	});

	it('infers icon by path or label when icon is omitted', () => {
		expect(resolveNavIcon({ label: 'Ops', href: '/ops' })).toBe(ActivityIcon);
		expect(resolveNavIcon({ label: 'Monitor', href: '/monitor' })).toBe(ScanSearchIcon);
		expect(resolveNavIcon({ label: 'Unknown', href: '/custom' })).toBe(LayersIcon);
	});
});

describe('activeNavHref', () => {
	it('matches Overview on the home page only', () => {
		expect(activeNavHref('/')).toBe('/');
		expect(activeNavHref('/somewhere')).toBeUndefined();
	});

	it('matches Monitor', () => {
		expect(activeNavHref('/monitor')).toBe('/monitor');
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
		expect(comingSoonSection('/monitor')).toBeUndefined();
		expect(comingSoonSection('/spreadsheet')).toBeUndefined();
	});
});
