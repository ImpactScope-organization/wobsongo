<script lang="ts">
	import { page } from '$app/state';
	import logo from '#lib/assets/ws-logo-white.svg';
	import AppHeader from '#lib/components/common/app-header.svelte';
	import { activeNavHref, NAV_ITEMS, SETTINGS_HREF } from '#lib/navigation.js';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
</script>

<div class="min-h-screen bg-background pb-20 md:pb-0">
	<AppHeader
		logoSrc={logo}
		items={NAV_ITEMS}
		currentHref={activeNavHref(page.url.pathname) ?? ''}
		account={{ name: data.account.name, href: SETTINGS_HREF }}
		mobileDock
	/>
	{@render children()}
</div>
