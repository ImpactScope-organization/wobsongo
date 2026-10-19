import type { Component } from 'svelte';
import ActivityIcon from '@lucide/svelte/icons/activity';
import LayersIcon from '@lucide/svelte/icons/layers';
import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
import MessageSquareTextIcon from '@lucide/svelte/icons/message-square-text';
import Radar from '@lucide/svelte/icons/radar';
import ShieldAlertIcon from '@lucide/svelte/icons/shield-alert';

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

export const NAV_ITEMS: readonly NavItem[] = [
	{ label: 'Overview', href: '/', icon: LayoutDashboardIcon },
	{ label: 'Narratives', href: '/narratives', disabled: true, icon: MessageSquareTextIcon },
	{ label: 'Spread', href: '/spread', disabled: true, icon: Radar },
	{ label: 'Operations', href: '/ops', disabled: true, icon: ActivityIcon },
	{ label: 'High-harm', href: '/safety', disabled: true, icon: ShieldAlertIcon }
];

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
	hrefs: readonly string[] = [...NAV_ITEMS.map((item) => item.href), SETTINGS_HREF]
): string | undefined {
	return hrefs
		.filter((href) =>
			href === '/' ? pathname === '/' : pathname === href || pathname.startsWith(`${href}/`)
		)
		.toSorted((a, b) => b.length - a.length)[0];
}

/** The nav item for a view that isn't built yet, if `pathname` is in one. */
export function comingSoonSection(pathname: string): NavItem | undefined {
	return NAV_ITEMS.find(
		(item) => item.disabled && (pathname === item.href || pathname.startsWith(`${item.href}/`))
	);
}
