<script lang="ts">
	import { Chart as LayerChart, Circle, Spline, Svg } from 'layerchart';
	import * as Chart from '#lib/components/ui/chart/index.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** Values oldest first. Fewer than two values draws nothing. */
		values: readonly number[];
		/** Accessible description, e.g. "Volume over the last 24 hours, rising". */
		label: string;
		/** `signal` draws the line in the attention colour. */
		tone?: 'default' | 'muted' | 'signal';
		/** Mark the latest value with a dot. */
		endDot?: boolean;
		width?: number;
		height?: number;
		class?: string;
	}

	let {
		values,
		label,
		tone = 'default',
		endDot = false,
		width = 80,
		height = 24,
		class: className
	}: Props = $props();

	const TONE_COLOR = {
		default: 'var(--foreground)',
		muted: 'var(--faint-foreground)',
		signal: 'var(--signal)'
	} as const;

	const data = $derived(values.map((value, index) => ({ index, value })));
	const last = $derived(data.at(-1));
	const config = $derived({
		value: { label, color: TONE_COLOR[tone] }
	}) satisfies Chart.ChartConfig;
</script>

<!-- Fixed size so it renders on the server, without JavaScript. -->
<Chart.Container
	{config}
	class={cn('aspect-auto shrink-0', className)}
	style="width: {width}px; height: {height}px"
	role="img"
	aria-label={label}
>
	{#if data.length >= 2}
		<!-- Built from primitives: LineChart's Layer wrapper crashes when rendered on the server. -->
		<LayerChart
			ssr
			{width}
			{height}
			{data}
			x="index"
			y="value"
			yDomain={[0, null]}
			padding={{ top: 3, right: endDot ? 3 : 1, bottom: 2, left: 1 }}
		>
			{#snippet children({ context })}
				<Svg>
					<Spline class="stroke-(--color-value) stroke-[1.5]" fill="none" />
					{#if endDot && last}
						<Circle
							cx={context.xScale(last.index)}
							cy={context.yScale(last.value)}
							r={2.5}
							class="fill-signal stroke-none"
						/>
					{/if}
				</Svg>
			{/snippet}
		</LayerChart>
	{/if}
</Chart.Container>
