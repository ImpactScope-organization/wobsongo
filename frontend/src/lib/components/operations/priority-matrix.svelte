<script lang="ts">
	import { Chart as LayerChart, Svg } from 'layerchart';
	import * as Chart from '#lib/components/ui/chart/index.js';
	import {
		PRIORITY_HARM_SPLIT,
		PRIORITY_VELOCITY_SPLIT,
		priorityQuadrant,
		prioritySeverity,
		type PriorityPoint,
		type PrioritySeverity
	} from '#lib/domain/operations.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatNumber } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { PRIORITY_QUADRANT_LABEL as QUADRANT_LABEL } from '../common/labels';

	interface Props {
		/** Numbered in this order, in the chart and the list below it. */
		points: readonly PriorityPoint[];
		hrefFor?: (id: string) => string;
		emptyText?: string;
		class?: string;
	}

	let {
		points,
		hrefFor = (id) => `/narratives/${encodeURIComponent(id)}`,
		emptyText = m.operations_matrix_empty(),
		class: className
	}: Props = $props();

	// Drawn at a fixed size (so it renders on the server) and scaled by viewBox,
	// like the spread timeline.
	const WIDTH = 480;
	const HEIGHT = 300;

	// Velocity runs 0 to 1; the padding keeps the biggest bubble (radius 21 of
	// 480) inside the chart at either end.
	const X_DOMAIN: [number, number] = [-0.05, 1.05];

	const quadrant = priorityQuadrant;
	const TONE: Record<PrioritySeverity, { fill: string; text: string }> = {
		high: { fill: 'fill-signal', text: 'text-signal' },
		elevated: {
			fill: 'fill-verdict-misleading-foreground',
			text: 'text-verdict-misleading-foreground'
		},
		low: { fill: 'fill-muted-foreground', text: 'text-muted-foreground' }
	};
	const tone = (point: PriorityPoint) => TONE[prioritySeverity(point)];

	/** Spoken after the title; the chart itself is not navigable. */
	function details(point: PriorityPoint): string {
		const velocity = Math.round(point.velocity * 100);
		const text = m.operations_matrix_details({
			quadrant: QUADRANT_LABEL[quadrant(point)],
			harm: point.harm,
			velocity,
			volume: formatNumber(point.volume7d)
		});
		return ` ${text}`;
	}

	const maxVolume = $derived(Math.max(1, ...points.map((point) => point.volume7d)));
	/** Area scales with volume: radius 8 to 21. */
	const radius = (point: PriorityPoint) => 8 + 13 * Math.sqrt(point.volume7d / maxVolume);

	const description = $derived.by(() => {
		const urgent = points.filter((point) => quadrant(point) === 'urgent');
		return [
			m.operations_matrix_description({ count: points.length }),
			urgent.length
				? m.operations_matrix_urgent({ titles: urgent.map((point) => point.title).join('; ') })
				: m.operations_matrix_none_urgent()
		].join(' ');
	});

	const config = {} satisfies Chart.ChartConfig;
</script>

{#if points.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<div class={cn('flex flex-col gap-3', className)}>
		<div class="flex gap-2">
			<span
				class="rotate-180 text-center font-mono text-[0.6875rem] font-medium text-faint-foreground [writing-mode:vertical-rl]"
				aria-hidden="true"
			>
				{m.operations_axis_harm()}
			</span>
			<div class="min-w-0 flex-1">
				<Chart.Container
					{config}
					class="aspect-[480/300] w-full overflow-hidden rounded-md font-mono [&_.lc-layout-svg]:inset-0! [&_.lc-layout-svg]:size-full! [&_.lc-root-container]:size-full! [&_.lc-tooltip-context]:inset-0! [&_.lc-tooltip-context]:size-full! [&_.lc-tooltip-context-container]:inset-0! [&_.lc-tooltip-context-container]:size-full!"
					role="img"
					aria-label={description}
				>
					<LayerChart
						ssr
						width={WIDTH}
						height={HEIGHT}
						data={[...points]}
						x="velocity"
						y="harm"
						xDomain={X_DOMAIN}
						yDomain={[0.5, 5.5]}
					>
						{#snippet children({ context })}
							{@const midX = context.xScale(PRIORITY_VELOCITY_SPLIT)}
							{@const midY = context.yScale(PRIORITY_HARM_SPLIT - 0.5)}
							{@const w = context.width}
							{@const h = context.height}
							<Svg viewBox="0 0 {WIDTH} {HEIGHT}">
								<rect x={0} y={0} width={midX - 1} height={midY - 1} class="fill-muted" />
								<rect
									x={midX + 1}
									y={0}
									width={w - midX - 1}
									height={midY - 1}
									class="fill-verdict-false"
								/>
								<rect
									x={0}
									y={midY + 1}
									width={midX - 1}
									height={h - midY - 1}
									class="fill-background"
								/>
								<rect
									x={midX + 1}
									y={midY + 1}
									width={w - midX - 1}
									height={h - midY - 1}
									class="fill-muted"
								/>

								<g class="text-[10.5px] font-semibold tracking-[0.04em] uppercase">
									<text x={10} y={20} class="fill-faint-foreground">{QUADRANT_LABEL.critical}</text>
									<text x={w - 10} y={20} text-anchor="end" class="fill-verdict-false-foreground">
										{QUADRANT_LABEL.urgent}
									</text>
									<text x={10} y={h - 10} class="fill-faint-foreground"
										>{QUADRANT_LABEL.monitor}</text
									>
									<text x={w - 10} y={h - 10} text-anchor="end" class="fill-faint-foreground">
										{QUADRANT_LABEL.campaign}
									</text>
								</g>

								{#each points as point, index (point.id)}
									{@const cx = context.xScale(point.velocity)}
									{@const cy = context.yScale(point.harm)}
									<g data-quadrant={quadrant(point)}>
										<circle
											{cx}
											{cy}
											r={radius(point)}
											class={cn('stroke-card stroke-2 opacity-85', tone(point).fill)}
										/>
										<text
											x={cx}
											y={cy}
											dy="0.35em"
											text-anchor="middle"
											class="fill-white text-[10px] font-semibold"
										>
											{index + 1}
										</text>
									</g>
								{/each}
							</Svg>
						{/snippet}
					</LayerChart>
				</Chart.Container>
				<p
					class="mt-2 text-center font-mono text-[0.6875rem] font-medium text-faint-foreground"
					aria-hidden="true"
				>
					{m.operations_axis_velocity()}
				</p>
			</div>
		</div>

		<ol class="flex flex-col gap-1 text-xs text-verdict-unverified-foreground">
			{#each points as point, index (point.id)}
				<li class="flex gap-2">
					<span
						class={cn('w-4 shrink-0 font-mono text-[0.6875rem] font-semibold', tone(point).text)}
						aria-hidden="true">{index + 1}</span
					>
					<span>
						{#if point.narrativeId}
							<a href={hrefFor(point.narrativeId)} class="hover:underline">{point.title}</a>
						{:else}
							{point.title}
						{/if}
						<span class="sr-only">{details(point)}</span>
					</span>
				</li>
			{/each}
		</ol>
	</div>
{/if}
