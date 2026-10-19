<script lang="ts">
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ChevronUpIcon from '@lucide/svelte/icons/chevron-up';
	import type { Anomaly } from '#lib/domain/narrative.js';
	import { formatPercent } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		anomaly: Anomaly;
		/** Link to the narrative. */
		href: string;
		class?: string;
		/** Force mobile layout (e.g. for testing / stories). */
		forceMobile?: boolean;
	}

	let { anomaly, href, class: className, forceMobile = false }: Props = $props();

	let expanded = $state(false);
</script>

<aside
	aria-label="Anomaly"
	class={cn(
		'mx-3.5 mt-2.5 rounded-xl border border-signal-foreground/25 bg-signal p-3 text-signal-foreground md:mx-0 md:mt-0 md:rounded-none md:border-0 md:px-[clamp(1rem,3vw,2rem)] md:py-0',
		forceMobile && 'mx-3.5 mt-2.5 rounded-xl border border-signal-foreground/25 p-3',
		className
	)}
>
	<!-- Mobile collapsible view (< md or forceMobile) -->
	<div class={forceMobile ? 'block' : 'md:hidden'}>
		{#if !expanded}
			<div
				class="flex cursor-pointer items-center gap-2.5 select-none"
				onclick={() => (expanded = true)}
				role="button"
				tabindex="0"
				onkeydown={(e) => {
					if (e.key === 'Enter' || e.key === ' ') {
						e.preventDefault();
						expanded = true;
					}
				}}
				aria-expanded={false}
				aria-label="Expand anomaly notice"
			>
				<span class="relative flex size-3">
					<span
						class="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-200 opacity-100"
					></span>
					<span class="relative inline-flex size-3 rounded-full border-2 border-white bg-red-500"
					></span>
				</span>
				<div class="min-w-0 flex-1 truncate text-xs/snug font-medium">
					<strong class="font-semibold">“{anomaly.narrativeTitle}”</strong>
					is spiking: {anomaly.sigma.toFixed(1)}σ above the usual level for this hour,
					{formatPercent(anomaly.growthPct, { signed: true })} in {anomaly.windowHours} h.
				</div>
				<span class="flex size-6 shrink-0 items-center justify-center rounded">
					<ChevronDownIcon class="size-4" aria-hidden="true" />
				</span>
			</div>
		{:else}
			<div class="flex items-start gap-2.5">
				<span class="relative mt-1 flex size-3">
					<span
						class="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-200 opacity-100"
					></span>
					<span class="relative inline-flex size-3 rounded-full border-2 border-white bg-red-500"
					></span>
				</span>
				<div class="flex-1 text-xs/relaxed font-medium text-pretty">
					<strong class="font-semibold">“{anomaly.narrativeTitle}”</strong>
					is spiking: {anomaly.sigma.toFixed(1)}σ above the usual level for this hour,
					{formatPercent(anomaly.growthPct, { signed: true })} in {anomaly.windowHours} h.
				</div>
			</div>
			<div class="mt-2.5 flex items-center justify-end gap-3 pt-0.5">
				<a
					{href}
					aria-label="Open narrative"
					class="ml-auto rounded-md bg-white px-3 py-[0.4375rem] text-xs font-semibold text-signal outline-none hover:bg-white/90 focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-signal"
				>
					Check <span aria-hidden="true">→</span>
				</a>
				<button
					type="button"
					onclick={() => (expanded = false)}
					aria-expanded={true}
					aria-label="Collapse anomaly notice"
					class="flex size-6 shrink-0 items-center justify-center rounded transition-colors hover:bg-white/10 focus-visible:ring-1 focus-visible:ring-current focus-visible:outline-none active:bg-white/20"
				>
					<ChevronUpIcon class="size-4" aria-hidden="true" />
				</button>
			</div>
		{/if}
	</div>

	<!-- Desktop static banner view (>= md unless forceMobile) -->
	<div
		class={cn(
			'mx-auto max-w-[85rem] items-center gap-3.5 py-2.5',
			forceMobile ? 'hidden' : 'hidden md:flex'
		)}
	>
		<div class="text-eyebrow flex items-center gap-2 font-semibold" aria-hidden="true">
			<span class="relative flex size-3">
				<span
					class="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-200 opacity-100 mix-blend-screen"
				></span>
				<span class="relative inline-flex size-3 rounded-full border-2 border-white bg-red-500"
				></span>
			</span>
		</div>
		<div
			class="line-clamp-1 min-w-60 flex-1 truncate text-sm/snug font-medium text-pretty md:line-clamp-2"
		>
			<strong class="font-semibold">“{anomaly.narrativeTitle}”</strong>
			is spiking: {anomaly.sigma.toFixed(1)}σ above the usual level for this hour,
			{formatPercent(anomaly.growthPct, { signed: true })} in {anomaly.windowHours} h.
		</div>
		<a
			{href}
			aria-label="Open narrative"
			class="ml-auto rounded-md bg-white px-3 py-[0.4375rem] text-xs font-semibold text-signal outline-none hover:bg-white/90 focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-signal"
		>
			Check <span aria-hidden="true">→</span>
		</a>
	</div>
</aside>
