<script lang="ts">
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ChevronUpIcon from '@lucide/svelte/icons/chevron-up';
	import type { Anomaly } from '#lib/domain/narrative.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatPercent, intlLocale } from '#lib/utils/format.js';
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

	const spiking = $derived(
		m.common_anomaly_spiking({
			sigma: anomaly.sigma.toLocaleString(intlLocale(), {
				minimumFractionDigits: 1,
				maximumFractionDigits: 1
			}),
			growth: formatPercent(anomaly.growthPct, { signed: true }),
			hours: anomaly.windowHours
		})
	);
</script>

<aside
	aria-label={m.common_anomaly_label()}
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
				aria-label={m.common_anomaly_expand()}
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
					{spiking}
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
					{spiking}
				</div>
			</div>
			<div class="mt-2.5 flex items-center justify-end gap-3 pt-0.5">
				<a
					{href}
					aria-label={m.common_open_narrative()}
					class="ml-auto rounded-md bg-white px-3 py-[0.4375rem] text-xs font-semibold text-signal outline-none hover:bg-white/90 focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-signal"
				>
					{m.common_check()} <span aria-hidden="true">→</span>
				</a>
				<button
					type="button"
					onclick={() => (expanded = false)}
					aria-expanded={true}
					aria-label={m.common_anomaly_collapse()}
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
			{spiking}
		</div>
		<a
			{href}
			aria-label={m.common_open_narrative()}
			class="ml-auto rounded-md bg-white px-3 py-[0.4375rem] text-xs font-semibold text-signal outline-none hover:bg-white/90 focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-signal"
		>
			{m.common_check()} <span aria-hidden="true">→</span>
		</a>
	</div>
</aside>
