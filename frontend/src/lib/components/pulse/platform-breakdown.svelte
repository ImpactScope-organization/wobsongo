<script lang="ts">
	import { Arc, Chart as LayerChart, Group, Pie, Svg } from 'layerchart';
	import * as Chart from '#lib/components/ui/chart/index.js';
	import type { Platform, PlatformCount } from '#lib/domain/platform.js';
	import { formatNumber, percentShares } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { PLATFORM_LABEL } from '../common/labels';

	interface Props {
		counts: readonly PlatformCount[];
		/** Donut diameter in px; fixed so it renders on the server. */
		size?: number;
		class?: string;
	}

	let { counts, size = 140, class: className }: Props = $props();

	const config = {
		whatsapp: { label: PLATFORM_LABEL.whatsapp, color: 'var(--platform-whatsapp)' },
		facebook: { label: PLATFORM_LABEL.facebook, color: 'var(--platform-facebook)' },
		tiktok: { label: PLATFORM_LABEL.tiktok, color: 'var(--platform-tiktok)' },
		x: { label: PLATFORM_LABEL.x, color: 'var(--platform-x)' },
		other: { label: PLATFORM_LABEL.other, color: 'var(--platform-other)' }
	} satisfies Record<Platform, { label: string; color: string }> & Chart.ChartConfig;

	const total = $derived(counts.reduce((sum, item) => sum + item.count, 0));
	const shares = $derived(percentShares(counts.map((item) => item.count)));
	const rows = $derived(
		counts.map((item, index) => ({
			...item,
			share: shares[index],
			color: config[item.platform].color
		}))
	);
	const description = $derived(
		rows.map((row) => `${config[row.platform].label} ${row.share}%`).join(', ')
	);
</script>

<div class={cn('flex flex-wrap items-center gap-5', className)}>
	<Chart.Container
		{config}
		class="relative aspect-square shrink-0"
		style="width: {size}px; height: {size}px"
		role="img"
		aria-label={total > 0 ? `Claims by platform: ${description}` : 'No claims by platform yet'}
	>
		{#if total > 0}
			<!-- Built from primitives: PieChart's Layer wrapper crashes when rendered on the server. -->
			<LayerChart ssr width={size} height={size} data={rows} x="count">
				<Svg>
					<Group center>
						<!-- Slice sizes come from the chart's `x` accessor ("count"). -->
						<Pie sort={null} innerRadius={-size * 0.15} padAngle={0.01}>
							{#snippet children({ arcs })}
								{#each arcs as arc (arc.data.platform)}
									<Arc
										startAngle={arc.startAngle}
										endAngle={arc.endAngle}
										padAngle={arc.padAngle}
										innerRadius={-size * 0.15}
										fill={arc.data.color}
									/>
								{/each}
							{/snippet}
						</Pie>
					</Group>
				</Svg>
			</LayerChart>
		{/if}
		<div
			class="pointer-events-none absolute inset-0 grid place-items-center text-center"
			aria-hidden="true"
		>
			<div>
				<div class="font-mono text-xl font-semibold">{formatNumber(total)}</div>
				<div class="text-[0.65625rem] font-medium text-muted-foreground">claims</div>
			</div>
		</div>
	</Chart.Container>

	<table class="min-w-[9.5rem] flex-1 text-[0.8125rem]">
		<caption class="sr-only">Incoming claims by source platform</caption>
		<thead class="sr-only">
			<tr><th scope="col">Platform</th><th scope="col">Claims</th><th scope="col">Share</th></tr>
		</thead>
		<tbody>
			{#each rows as row (row.platform)}
				<tr>
					<th scope="row" class="py-1 text-left font-medium">
						<span class="flex items-center gap-2">
							<span class="size-2.5 shrink-0 rounded-[2px]" style:background={row.color}></span>
							{config[row.platform].label}
						</span>
					</th>
					<td class="py-1 text-right font-mono text-xs text-muted-foreground">
						{formatNumber(row.count)}
					</td>
					<td class="w-10 py-1 text-right font-mono text-xs font-semibold">{row.share}%</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
