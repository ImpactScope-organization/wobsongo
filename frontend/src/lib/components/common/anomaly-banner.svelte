<script lang="ts">
	import type { Anomaly } from '#lib/domain/narrative.js';
	import { formatPercent } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		anomaly: Anomaly;
		/** Link to the narrative. */
		href: string;
		class?: string;
	}

	let { anomaly, href, class: className }: Props = $props();
</script>

<aside
	aria-label="Anomaly"
	class={cn('bg-signal px-[clamp(1rem,3vw,2rem)] text-signal-foreground', className)}
>
	<div
		class="mx-auto flex max-w-[85rem] flex-col flex-wrap items-start gap-3.5 py-2.5 md:flex-row md:items-center"
	>
		<div class="text-eyebrow flex items-center gap-2 font-semibold" aria-hidden="true">
			<span class="size-2 rounded-full bg-current motion-safe:animate-pulse"></span>
			Anomaly
		</div>
		<div
			class="line-clamp-1 w-full min-w-60 flex-1 truncate text-sm/snug font-medium text-pretty md:line-clamp-2"
		>
			<strong class="font-semibold">“{anomaly.narrativeTitle}”</strong>
			is spiking: {anomaly.sigma.toFixed(1)}σ above the usual level for this hour,
			{formatPercent(anomaly.growthPct, { signed: true })} in {anomaly.windowHours} h.
		</div>
		<a
			{href}
			class="ml-auto rounded-md bg-white px-3 py-[0.4375rem] text-xs font-semibold text-signal outline-none hover:bg-white/90 focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-signal"
		>
			Open narrative <span aria-hidden="true">→</span>
		</a>
	</div>
</aside>
