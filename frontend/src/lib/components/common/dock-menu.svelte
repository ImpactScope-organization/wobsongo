<script lang="ts">
	import type { Component } from 'svelte';
	import type { NavItem } from './app-header.svelte';
	import { cn } from '#lib/utils.js';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import MessageSquareTextIcon from '@lucide/svelte/icons/message-square-text';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import ShieldAlertIcon from '@lucide/svelte/icons/shield-alert';
	import LayersIcon from '@lucide/svelte/icons/layers';
	import Radar from '@lucide/svelte/icons/radar';

	interface Props {
		items: readonly NavItem[];
		/** href of current section, matched against items */
		currentHref: string;
		class?: string;
	}

	let { items, currentHref, class: className }: Props = $props();

	function resolveIcon(item: NavItem): Component {
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
</script>

<nav
	aria-label="Mobile Navigation"
	class={cn(
		'fixed bottom-3 inset-x-3 z-50 mx-auto max-w-md md:hidden',
		className
	)}
>
	<div
		class="relative flex items-center justify-around gap-1 rounded-2xl border border-inverse-border/80 bg-inverse/95 p-1.5 shadow-[0_10px_35px_-5px_rgba(0,0,0,0.5),0_0_0_1px_rgba(255,255,255,0.05)] backdrop-blur-xl text-inverse-foreground"
	>
		{#each items as item (item.href)}
			{@const Icon = resolveIcon(item)}
			{@const isCurrent = item.href === currentHref}

			{#if item.disabled}
				<span
					aria-disabled="true"
					title="Coming soon"
					class="relative flex flex-1 flex-col items-center justify-center gap-1 rounded-xl py-1.5 px-1 text-center text-inverse-muted-foreground cursor-not-allowed select-none"
				>
					<span class="relative flex size-5 items-center justify-center">
						<Icon class="size-5 shrink-0 stroke-[1.75]" />
					</span>
					<span class="text-[0.625rem] font-medium leading-none tracking-tight truncate max-w-17">
						{item.label}
						<span class="sr-only">&nbsp;(coming soon)</span>
					</span>
				</span>
			{:else}
				<a
					href={item.href}
					aria-current={isCurrent ? 'page' : undefined}
					class={cn(
						'group relative flex flex-1 flex-col items-center justify-center gap-1 rounded-xl py-1.5 px-1 text-center outline-none transition-all duration-200 active:scale-95 focus-visible:ring-2 focus-visible:ring-inverse-signal',
						isCurrent
							? 'bg-inverse-raised text-inverse-signal font-semibold shadow-inner'
							: 'text-inverse-muted-foreground hover:bg-inverse-raised/60 hover:text-inverse-foreground'
					)}
				>
					<span class="relative flex size-5 items-center justify-center">
						<Icon
							class={cn(
								'size-5 shrink-0 transition-transform duration-200 group-hover:scale-110',
								isCurrent ? 'stroke-[2.25]' : 'stroke-[1.75]'
							)}
						/>
						{#if item.count}
							<span
								class="absolute -top-1.5 -right-2 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-signal px-1 font-mono text-[0.5625rem] font-bold text-signal-foreground shadow-sm"
							>
								{item.count}
								<span class="sr-only">&nbsp;open</span>
							</span>
						{/if}
					</span>
					<span class="text-[0.625rem] font-medium leading-none tracking-tight truncate max-w-17">
						{item.label}
					</span>
					{#if isCurrent}
						<span class="absolute -bottom-0.5 size-1 rounded-full bg-inverse-signal"></span>
					{/if}
				</a>
			{/if}
		{/each}
	</div>
</nav>
