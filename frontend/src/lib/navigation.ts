/** The main navigation. Views that aren't built yet are listed but disabled. */
import type { NavItem } from '#lib/components/common/app-header.svelte';

export const SETTINGS_HREF = '/settings';

export const NAV_ITEMS: readonly NavItem[] = [
	{ label: 'Overview', href: '/' },
	{ label: 'Narratives', href: '/narratives', disabled: true },
	{ label: 'Spread', href: '/spread', disabled: true },
	{ label: 'Operations', href: '/ops', disabled: true },
	{ label: 'High-harm', href: '/safety', disabled: true }
];

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
