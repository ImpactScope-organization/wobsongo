<script lang="ts" module>
	export interface StackedSegment {
		key: string;
		/** Used in the accessible description, e.g. "false" or "Voice note". */
		label: string;
		value: number;
		/** Background (and text) classes, e.g. "bg-primary text-primary-foreground". */
		class: string;
	}
</script>

<script lang="ts">
	import { formatNumber, percentShares } from '#lib/utils/format.js';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		segments: readonly StackedSegment[];
		/** How the description reads each segment: "14 false" or "Voice note 52%". */
		describe?: 'count' | 'percent';
		/** Print each segment's percentage inside it (not in the `sm` size). */
		showPercent?: boolean;
		/** `sm` is a thin meter; `md` a compact row bar; `lg` is tall enough for labels. */
		size?: 'sm' | 'md' | 'lg';
		emptyText?: string;
		class?: string;
	}

	let {
		segments,
		describe = 'count',
		showPercent = false,
		size = 'sm',
		emptyText = m.common_no_data(),
		class: className
	}: Props = $props();

	// A proportion bar rather than a chart: flex segments, no JavaScript.
	const shares = $derived(percentShares(segments.map((segment) => segment.value)));
	const visible = $derived(
		segments
			.map((segment, index) => ({ ...segment, share: shares[index] }))
			.filter((segment) => segment.value > 0)
	);
	const description = $derived(
		visible
			.map((segment) =>
				describe === 'count'
					? `${formatNumber(segment.value)} ${segment.label}`
					: `${segment.label} ${segment.share}%`
			)
			.join(', ') || emptyText
	);
</script>

<div
	role="img"
	aria-label={description}
	title={description}
	data-size={size}
	class={cn(
		'flex gap-0.5 overflow-hidden bg-secondary data-[size=lg]:h-7 data-[size=lg]:rounded-[0.3125rem] data-[size=md]:h-5 data-[size=md]:rounded-[3px] data-[size=sm]:h-1.5 data-[size=sm]:rounded-full',
		className
	)}
>
	{#each visible as segment (segment.key)}
		<div
			data-segment={segment.key}
			class={cn(
				'flex min-w-0 items-center overflow-hidden px-2 font-mono text-[0.6875rem] font-semibold',
				segment.class
			)}
			style:flex-grow={segment.value}
			style:flex-basis="0"
		>
			{#if showPercent && size !== 'sm' && segment.share >= 10}
				<span aria-hidden="true">{segment.share}%</span>
			{/if}
		</div>
	{/each}
</div>
