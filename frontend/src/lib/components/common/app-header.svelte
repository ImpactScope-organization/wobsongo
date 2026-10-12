<script lang="ts" module>
	export interface NavItem {
		label: string;
		href: string;
		/** Count badge, e.g. open high-harm alerts. Hidden when 0 or absent. */
		count?: number;
	}
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '#lib/utils.js';

	interface Props {
		productName?: string;
		/** Logo image URL; omitted shows the name only. */
		logoSrc?: string;
		items: readonly NavItem[];
		/** href of the current section, matched against `items`. */
		currentHref: string;
		/** Right-hand controls, e.g. an export link. */
		actions?: Snippet;
		class?: string;
	}

	let {
		productName = 'WobSongo Monitor',
		logoSrc,
		items,
		currentHref,
		actions,
		class: className
	}: Props = $props();
</script>

<header class={cn('bg-inverse px-[clamp(1rem,3vw,2rem)] text-inverse-foreground', className)}>
	<div class="mx-auto flex min-h-14 max-w-[85rem] flex-wrap items-center gap-x-5 gap-y-2 py-2">
		<a
			href="/"
			class="flex items-center gap-2.5 rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-inverse-signal"
		>
			{#if logoSrc}
				<img
					src={logoSrc}
					alt=""
					width="30"
					height="30"
					class="size-[1.875rem] rounded-md bg-white"
				/>
			{/if}
			<span class="font-heading text-[0.9375rem] leading-none font-bold tracking-[-0.01em]">
				{productName}
			</span>
		</a>

		<nav
			aria-label="Main"
			class="order-last min-w-0 basis-full overflow-x-auto sm:order-none sm:flex-1 sm:basis-auto"
		>
			<ul class="flex gap-0.5">
				{#each items as item (item.href)}
					<li>
						<a
							href={item.href}
							aria-current={item.href === currentHref ? 'page' : undefined}
							class="flex items-center gap-1.5 rounded-md px-3 py-2 text-[0.8125rem] font-medium whitespace-nowrap transition-colors outline-none hover:bg-inverse-raised focus-visible:ring-2 focus-visible:ring-inverse-signal aria-[current=page]:bg-inverse-border"
						>
							{item.label}
							{#if item.count}
								<span
									class="rounded-full bg-signal px-1.5 py-px font-mono text-[0.625rem] font-semibold text-signal-foreground"
								>
									{item.count}<span class="sr-only">&nbsp;open</span>
								</span>
							{/if}
						</a>
					</li>
				{/each}
			</ul>
		</nav>

		{#if actions}
			<div class="ml-auto flex items-center gap-2.5 sm:ml-0">{@render actions()}</div>
		{/if}
	</div>
</header>
