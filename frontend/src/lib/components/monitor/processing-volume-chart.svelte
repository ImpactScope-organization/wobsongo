<script lang="ts">
	import { successRate, type VolumeBucket } from '#lib/domain/monitor.js';
	import { m } from '#lib/paraglide/messages.js';
	import {
		formatClockTime,
		formatDayMonth,
		formatNumber,
		formatPercent
	} from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import SwatchLegend from '../common/swatch-legend.svelte';

	interface Props {
		/** Oldest first. */
		buckets: readonly VolumeBucket[];
		/** Length of each bucket; under 24 labels buckets by clock time, else by day. */
		bucketHours: number;
		class?: string;
	}

	let { buckets, bucketHours, class: className }: Props = $props();

	const HOUR = 60 * 60 * 1000;

	function label(bucket: VolumeBucket): string {
		if (bucketHours >= 24) return formatDayMonth(bucket.start);
		const end = new Date(bucket.start.getTime() + bucketHours * HOUR);
		return `${formatClockTime(bucket.start)}–${formatClockTime(end)}`;
	}

	const max = $derived(Math.max(1, ...buckets.map((bucket) => bucket.processed + bucket.failed)));
	const columns = $derived(
		buckets.map((bucket) => {
			const rate = successRate(bucket);
			return {
				...bucket,
				key: bucket.start.toISOString(),
				label: label(bucket),
				rate: rate === null ? '—' : formatPercent(rate)
			};
		})
	);
	const axis = $derived(
		buckets.length === 0
			? []
			: [...new Set([0, Math.floor((buckets.length - 1) / 2), buckets.length - 1])].map((index) =>
					bucketHours >= 24
						? formatDayMonth(buckets[index].start)
						: formatClockTime(buckets[index].start)
				)
	);

	// The readout above the bars shows one bucket: the one under the pointer or
	// finger, else the one last tapped or picked with the keyboard, else the
	// newest. It sits inside the card, so it can't be clipped, and without
	// JavaScript it simply shows the newest bucket.
	let pinned = $state<number | null>(null);
	let hovered = $state<number | null>(null);
	const last = $derived(columns.length - 1);
	const clamp = (index: number) => Math.min(Math.max(index, 0), last);
	const active = $derived(clamp(hovered ?? pinned ?? last));
	const shown = $derived(columns[active]);
	const inspecting = $derived(hovered !== null || pinned !== null);

	let bars = $state<HTMLElement>();

	/** The bar under a pointer, from its x position, so thin bars are easy to hit and scrub. */
	function barAt(event: PointerEvent): number {
		const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
		return clamp(Math.floor(((event.clientX - rect.left) / rect.width) * columns.length));
	}

	function pick(index: number) {
		pinned = clamp(index);
		bars?.querySelectorAll('button')[pinned]?.focus();
	}

	function onkeydown(event: KeyboardEvent) {
		const moves: Record<string, number> = {
			ArrowLeft: active - 1,
			ArrowRight: active + 1,
			Home: 0,
			End: last
		};
		if (!(event.key in moves)) return;
		event.preventDefault();
		hovered = null;
		pick(moves[event.key]);
	}
</script>

{#if buckets.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>
		{m.common_no_data_in_window()}
	</p>
{:else}
	<figure class={cn('flex flex-col', className)}>
		<!-- Repeats what the focused bar's label says, so it's hidden from screen readers. -->
		<div
			aria-hidden="true"
			class="mb-3 flex flex-wrap items-center gap-x-4 gap-y-1 rounded-md bg-muted px-3 py-2 text-xs font-medium"
		>
			<span class="font-mono font-semibold">{shown.label}</span>
			<span class="flex items-center gap-1.5">
				<span class="size-2 rounded-[2px] bg-primary"></span>
				{formatNumber(shown.processed)}
				{m.monitor_volume_processed().toLowerCase()}
			</span>
			<span class="flex items-center gap-1.5">
				<span class="size-2 rounded-[2px] bg-signal"></span>
				{formatNumber(shown.failed)}
				{m.monitor_volume_failed().toLowerCase()}
			</span>
			<span class="font-mono text-muted-foreground">
				{m.monitor_volume_success({ rate: shown.rate })}
			</span>
		</div>

		<!-- A toolbar of bars: one tab stop for the chart, arrow keys move between bars. -->
		<div
			bind:this={bars}
			role="toolbar"
			tabindex="-1"
			aria-label={m.monitor_volume_caption()}
			class="flex h-[10.625rem] touch-pan-y items-end gap-[3px] border-b"
			onpointermove={(event) => (hovered = barAt(event))}
			onpointerleave={() => (hovered = null)}
			onpointerup={(event) => {
				// A tap or scrub on a touch screen keeps the bar it ended on.
				if (event.pointerType !== 'mouse') pinned = barAt(event);
			}}
			{onkeydown}
		>
			{#each columns as column, index (column.key)}
				<button
					type="button"
					tabindex={index === active ? 0 : -1}
					aria-pressed={index === active}
					aria-label={m.monitor_volume_bar({
						period: column.label,
						processed: formatNumber(column.processed),
						failed: formatNumber(column.failed),
						rate: column.rate
					})}
					data-active={inspecting && index === active ? true : undefined}
					data-dimmed={inspecting && index !== active ? true : undefined}
					class="flex h-full min-w-0 flex-1 cursor-pointer flex-col justify-end outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 data-dimmed:opacity-45"
					onclick={() => pick(index)}
				>
					{#if column.failed > 0}
						<span
							data-segment="failed"
							class="block rounded-t-[2px] bg-signal"
							style:height="{(column.failed / max) * 100}%"
						></span>
					{/if}
					<span
						data-segment="processed"
						class="block bg-primary"
						style:height="{(column.processed / max) * 100}%"
					></span>
				</button>
			{/each}
		</div>
		<div
			aria-hidden="true"
			class="mt-1.5 flex justify-between font-mono text-[0.65625rem] font-medium text-faint-foreground"
		>
			{#each axis as tick, index (index)}
				<span>{tick}</span>
			{/each}
		</div>

		<SwatchLegend
			class="mt-3"
			items={[
				{ key: 'processed', label: m.monitor_volume_processed(), class: 'bg-primary' },
				{ key: 'failed', label: m.monitor_volume_failed(), class: 'bg-signal' }
			]}
		/>
	</figure>
{/if}
