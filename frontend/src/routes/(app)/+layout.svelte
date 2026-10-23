<script lang="ts">
	import { page } from '$app/state';
	import logo from '#lib/assets/ws-logo-white.svg';
	import AppHeader from '#lib/components/common/app-header.svelte';
	import { activeNavHref, navItems, SETTINGS_HREF } from '#lib/navigation.js';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
</script>

<div class="min-h-screen bg-background pb-20 md:pb-0">
	<AppHeader
		logoSrc={logo}
		items={navItems()}
		currentHref={activeNavHref(page.url.pathname) ?? ''}
		account={{ name: data.account.name, href: SETTINGS_HREF }}
		returnTo={page.url.pathname + page.url.search}
		mobileDock
	/>
	{@render children()}
</div>
