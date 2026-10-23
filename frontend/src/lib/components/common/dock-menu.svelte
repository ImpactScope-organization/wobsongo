<script lang="ts">
	import { resolveNavIcon, type NavItem } from '#lib/navigation.js';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		items: readonly NavItem[];
		/** href of current section, matched against items */
		currentHref: string;
		class?: string;
	}

	let { items, currentHref, class: className }: Props = $props();
</script>

<nav
	aria-label={m.nav_mobile_label()}
	class={cn('fixed inset-x-3 bottom-3 z-50 mx-auto max-w-md md:hidden', className)}
>
	<div
		class="relative flex items-center justify-around gap-1 rounded-2xl border border-inverse-border/80 bg-inverse/95 p-1.5 text-inverse-foreground shadow-[0_10px_35px_-5px_rgba(0,0,0,0.5),0_0_0_1px_rgba(255,255,255,0.05)] backdrop-blur-xl"
	>
		{#each items as item (item.href)}
			{@const Icon = resolveNavIcon(item)}
			{@const isCurrent = item.href === currentHref}

			{#if item.disabled}
				<span
					aria-disabled="true"
					title={m.nav_coming_soon()}
					class="relative flex flex-1 cursor-not-allowed flex-col items-center justify-center gap-1 rounded-xl px-1 py-1.5 text-center text-inverse-muted-foreground select-none"
				>
					<span class="relative flex size-5 items-center justify-center">
						<Icon class="size-5 shrink-0 stroke-[1.75]" />
					</span>
					<span class="max-w-17 truncate text-[0.625rem] leading-none font-medium tracking-tight">
						{item.label}
						<span class="sr-only">&nbsp;{m.nav_coming_soon_suffix()}</span>
					</span>
				</span>
			{:else}
				<a
					href={item.href}
					aria-current={isCurrent ? 'page' : undefined}
					class={cn(
						'group relative flex flex-1 flex-col items-center justify-center gap-1 rounded-xl px-1 py-1.5 text-center transition-all duration-200 outline-none focus-visible:ring-2 focus-visible:ring-inverse-signal active:scale-95',
						isCurrent
							? 'bg-inverse-raised font-semibold text-inverse-signal shadow-inner'
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
								<span class="sr-only">&nbsp;{m.nav_count_open()}</span>
							</span>
						{/if}
					</span>
					<span class="max-w-17 truncate text-[0.625rem] leading-none font-medium tracking-tight">
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
