<script lang="ts">
	import { Chart as LayerChart, Spline, Svg } from 'layerchart';
	import * as Chart from '#lib/components/ui/chart/index.js';
	import { whatsAppLagHours, type NarrativeSpread } from '#lib/domain/narrative-detail.js';
	import type { Platform } from '#lib/domain/platform.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatDayTime } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { PLATFORM_SHORT_LABEL } from '../common/labels';

	interface Props {
		spread: NarrativeSpread;
		/** End of the window, labelled "Now"; pass the injected clock's now. */
		now: Date;
		class?: string;
	}

	let { spread, now, class: className }: Props = $props();

	// Drawn at a fixed size (so it renders on the server) and scaled by viewBox:
	// the container classes stretch LayerChart's fixed-size wrappers to the box.
	const WIDTH = 640;
	const HEIGHT = 230;
	const PADDING = { top: 20, right: 10, bottom: 30, left: 20 };
	const DAY = 24 * 60 * 60 * 1000;

	const config = {
		whatsapp: { label: PLATFORM_SHORT_LABEL.whatsapp, color: 'var(--platform-whatsapp)' },
		facebook: { label: PLATFORM_SHORT_LABEL.facebook, color: 'var(--platform-facebook)' },
		tiktok: { label: PLATFORM_SHORT_LABEL.tiktok, color: 'var(--platform-tiktok)' },
		x: { label: PLATFORM_SHORT_LABEL.x, color: 'var(--platform-x)' },
		other: { label: PLATFORM_SHORT_LABEL.other, color: 'var(--platform-other)' }
	} satisfies Record<Platform, { label: string; color: string }> & Chart.ChartConfig;

	const points = $derived(spread.series.flatMap((series) => series.points));
	// Starts at the first sighting even if the series begin later, so its marker stays on the chart.
	const start = $derived(
		Math.min(spread.firstSeen.at.getTime(), ...points.map((point) => point.at.getTime()))
	);
	// At least 1, so all-zero series sit on the baseline instead of mid-height.
	const maxCount = $derived(Math.max(1, ...points.map((point) => point.count)));
	const lag = $derived(whatsAppLagHours(spread));

	/** Midnights (UTC) strictly inside the window, for x-axis labels. */
	const midnights = $derived.by(() => {
		const ticks: Date[] = [];
		for (let t = Math.ceil(start / DAY) * DAY; t < now.getTime(); t += DAY) ticks.push(new Date(t));
		return ticks;
	});

	const description = $derived(
		[
			m.narrative_timeline_mentions({
				platforms: spread.series.map((series) => config[series.platform].label).join(', ')
			}),
			m.narrative_timeline_first_seen_on({
				platform: config[spread.firstSeen.platform].label,
				time: formatDayTime(spread.firstSeen.at)
			}),
			spread.firstOnWhatsApp
				? m.narrative_timeline_reached_whatsapp({
						time: formatDayTime(spread.firstOnWhatsApp),
						lag: String(lag)
					})
				: m.narrative_timeline_not_on_whatsapp()
		].join(' ')
	);
</script>

<div class={cn('flex flex-col gap-3.5', className)}>
	<ul class="flex flex-wrap gap-3.5 text-xs font-medium" aria-hidden="true">
		{#each spread.series as series (series.platform)}
			<li class="flex items-center gap-1.5">
				<span class="h-0.5 w-3.5" style:background={config[series.platform].color}></span>
				{config[series.platform].label}
			</li>
		{/each}
	</ul>

	<Chart.Container
		{config}
		class="aspect-[640/230] w-full font-mono [&_.lc-layout-svg]:inset-0! [&_.lc-layout-svg]:size-full! [&_.lc-root-container]:size-full! [&_.lc-tooltip-context]:inset-0! [&_.lc-tooltip-context]:size-full! [&_.lc-tooltip-context-container]:inset-0! [&_.lc-tooltip-context-container]:size-full!"
		role="img"
		aria-label={description}
	>
		{#if points.length > 0}
			<!-- Built from primitives: LineChart's Layer wrapper crashes when rendered on the server. -->
			<LayerChart
				ssr
				width={WIDTH}
				height={HEIGHT}
				data={points}
				x="at"
				y="count"
				xDomain={[new Date(start), now]}
				yDomain={[0, maxCount]}
				yNice
				padding={PADDING}
			>
				{#snippet children({ context })}
					{@const x = (at: Date) => context.xScale(at)}
					{@const bottom = context.height}
					{@const first = x(spread.firstSeen.at)}
					{@const reached = spread.firstOnWhatsApp ? x(spread.firstOnWhatsApp) : null}
					<!-- Two label columns need ~110 units; closer than that, WhatsApp's drop a line. -->
					{@const labelY = reached !== null && reached - first < 110 ? 42 : 14}
					<Svg viewBox="0 0 {WIDTH} {HEIGHT}">
						<!-- Grid: baseline plus three dashed levels. -->
						{#each [0, 1 / 3, 2 / 3, 1] as level (level)}
							<line
								x1={0}
								x2={context.width}
								y1={bottom * (1 - level)}
								y2={bottom * (1 - level)}
								class={level === 0 ? 'stroke-border' : 'stroke-rule'}
								stroke-dasharray={level === 0 ? undefined : '2 3'}
							/>
						{/each}

						<!-- The gap between first public sighting and the bot. -->
						{#if reached !== null}
							<rect
								x={first}
								y={0}
								width={Math.max(0, reached - first)}
								height={bottom}
								class="fill-background"
							/>
							<line
								x1={reached}
								x2={reached}
								y1={0}
								y2={bottom}
								class="stroke-platform-whatsapp"
								stroke-dasharray="3 3"
							/>
						{/if}
						<line
							x1={first}
							x2={first}
							y1={0}
							y2={bottom}
							class="stroke-foreground"
							stroke-dasharray="3 3"
						/>

						{#each spread.series as series (series.platform)}
							<Spline
								data={series.points}
								x="at"
								y="count"
								fill="none"
								stroke={config[series.platform].color}
								class={series.platform === 'whatsapp' ? 'stroke-[2.5]' : 'stroke-2'}
							/>
						{/each}

						<circle cx={first} cy={bottom} r={4} fill={config[spread.firstSeen.platform].color} />
						{#if reached !== null}
							<circle cx={reached} cy={bottom} r={4} class="fill-platform-whatsapp" />
						{/if}

						<g class="text-[10.5px]">
							<text x={first + 6} y={14} class="fill-foreground"
								>{m.narrative_timeline_first_seen()}</text
							>
							<text x={first + 6} y={28} class="fill-muted-foreground">
								{config[spread.firstSeen.platform].label} · {formatDayTime(spread.firstSeen.at)}
							</text>
							{#if reached !== null && spread.firstOnWhatsApp}
								<text x={reached + 6} y={labelY} class="fill-platform-whatsapp">WhatsApp</text>
								<text x={reached + 6} y={labelY + 14} class="fill-muted-foreground">
									{formatDayTime(spread.firstOnWhatsApp)}
								</text>
								{#if reached - first >= 70}
									<text
										x={(first + reached) / 2}
										y={bottom / 2 - 6}
										text-anchor="middle"
										class="fill-foreground text-xs font-semibold"
									>
										+{lag} h
									</text>
									<text
										x={(first + reached) / 2}
										y={bottom / 2 + 8}
										text-anchor="middle"
										class="fill-muted-foreground text-[10px]"
									>
										{m.narrative_timeline_public_to_whatsapp()}
									</text>
								{/if}
							{/if}
						</g>

						<g class="fill-faint-foreground text-[10px]">
							{#each midnights as midnight (midnight.getTime())}
								<!-- Skip a midnight label that would collide with "Now". -->
								{#if x(midnight) < context.width - 70}
									<text x={x(midnight)} y={bottom + 20}>{formatDayTime(midnight)}</text>
								{/if}
							{/each}
							<text x={context.width} y={bottom + 20} text-anchor="end"
								>{m.narrative_timeline_now()}</text
							>
						</g>
					</Svg>
				{/snippet}
			</LayerChart>
		{/if}
	</Chart.Container>
</div>
