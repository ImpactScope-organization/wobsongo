import type { Component } from 'svelte';
import ActivityIcon from '@lucide/svelte/icons/activity';
import LayersIcon from '@lucide/svelte/icons/layers';
import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
import MessageSquareTextIcon from '@lucide/svelte/icons/message-square-text';
import Radar from '@lucide/svelte/icons/radar';
import ShieldAlertIcon from '@lucide/svelte/icons/shield-alert';
import { m } from '#lib/paraglide/messages.js';

export interface NavItem {
	label: string;
	href: string;
	/** Count badge, e.g. open high-harm alerts. Hidden when 0 or absent. */
	count?: number;
	/** Not built yet: shown, but not a link. */
	disabled?: boolean;
	/** Optional icon component for dock / navigation items. */
	icon?: Component;
}

export const SETTINGS_HREF = '/settings';

const NAV: readonly (Omit<NavItem, 'label'> & { label: () => string })[] = [
	{ label: m.nav_overview, href: '/', icon: LayoutDashboardIcon },
	{ label: m.nav_narratives, href: '/narratives', disabled: true, icon: MessageSquareTextIcon },
	{ label: m.nav_spread, href: '/spread', disabled: true, icon: Radar },
	{ label: m.nav_operations, href: '/ops', disabled: true, icon: ActivityIcon },
	{ label: m.nav_high_harm, href: '/safety', disabled: true, icon: ShieldAlertIcon }
];

/** The main nav, labelled in the current language. */
export function navItems(): NavItem[] {
	return NAV.map((item) => ({ ...item, label: item.label() }));
}

/**
 * Returns the icon component for a nav item, either from item.icon or inferred by href/label.
 */
export function resolveNavIcon(item: NavItem): Component {
	if (item.icon) return item.icon;
	const path = item.href.replace(/^\//, '').split('/')[0] || '';
	if (!path || path === 'overview') return LayoutDashboardIcon;
	if (path === 'narratives') return MessageSquareTextIcon;
	if (path === 'spread') return Radar;
	if (path === 'ops') return ActivityIcon;
	if (path === 'safety') return ShieldAlertIcon;

	const label = item.label.toLowerCase();
	if (label.includes('overview')) return LayoutDashboardIcon;
	if (label.includes('narrative')) return MessageSquareTextIcon;
	if (label.includes('spread')) return Radar;
	if (label.includes('operation') || label.includes('ops')) return ActivityIcon;
	if (label.includes('harm') || label.includes('safety')) return ShieldAlertIcon;

	return LayersIcon;
}

/**
 * The nav href to mark current for `pathname`: the longest one it is under.
 * "/" only matches itself, so every page doesn't light up Overview.
 */
export function activeNavHref(
	pathname: string,
	hrefs: readonly string[] = [...NAV.map((item) => item.href), SETTINGS_HREF]
): string | undefined {
	return hrefs
		.filter((href) =>
			href === '/' ? pathname === '/' : pathname === href || pathname.startsWith(`${href}/`)
		)
		.toSorted((a, b) => b.length - a.length)[0];
}

/** The nav item for a view that isn't built yet, if `pathname` is in one. */
export function comingSoonSection(pathname: string): NavItem | undefined {
	return navItems().find(
		(item) => item.disabled && (pathname === item.href || pathname.startsWith(`${item.href}/`))
	);
}
