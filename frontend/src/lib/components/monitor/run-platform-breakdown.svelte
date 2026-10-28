<script lang="ts">
	import { Arc, Chart as LayerChart, Group, Pie, Svg } from 'layerchart';
	import * as Chart from '#lib/components/ui/chart/index.js';
	import { CRAWLED_PLATFORMS, type CrawledPlatform } from '#lib/domain/monitor.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatNumber, percentShares } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { CRAWLED_PLATFORM_LABEL } from '../common/labels';

	interface Props {
		/** New posts by platform; null while the run is still going. */
		counts: Readonly<Record<CrawledPlatform, number>> | null;
		/** Donut diameter in px; fixed so it renders on the server. */
		size?: number;
		class?: string;
	}

	let { counts, size = 168, class: className }: Props = $props();

	const config = {
		facebook: { label: CRAWLED_PLATFORM_LABEL.facebook, color: 'var(--platform-facebook)' },
		tiktok: { label: CRAWLED_PLATFORM_LABEL.tiktok, color: 'var(--platform-tiktok)' },
		x: { label: CRAWLED_PLATFORM_LABEL.x, color: 'var(--platform-x)' },
		instagram: { label: CRAWLED_PLATFORM_LABEL.instagram, color: 'var(--platform-instagram)' },
		youtube: { label: CRAWLED_PLATFORM_LABEL.youtube, color: 'var(--platform-youtube)' }
	} satisfies Record<CrawledPlatform, { label: string; color: string }> & Chart.ChartConfig;

	const values = $derived(CRAWLED_PLATFORMS.map((platform) => counts?.[platform] ?? 0));
	const total = $derived(values.reduce((sum, count) => sum + count, 0));
	const shares = $derived(percentShares(values));
	const rows = $derived(
		CRAWLED_PLATFORMS.map((platform, index) => ({
			platform,
			count: values[index],
			share: shares[index],
			color: config[platform].color
		}))
	);
	const description = $derived(
		rows
			.filter((row) => row.count > 0)
			.map((row) => `${config[row.platform].label} ${row.share}%`)
			.join(', ')
	);
	const innerRadius = $derived(-size * 0.2);
</script>

{#snippet pending()}
	<span aria-hidden="true">—</span><span class="sr-only">{m.monitor_pending()}</span>
{/snippet}

<div class={cn('flex flex-wrap items-center gap-6', className)}>
	<Chart.Container
		{config}
		class="relative mx-auto aspect-square shrink-0"
		style="width: {size}px; height: {size}px"
		role="img"
		aria-label={total > 0
			? m.monitor_platform_chart({ total: formatNumber(total), description })
			: m.monitor_platform_chart_empty()}
	>
		{#if total > 0}
			<!-- Built from primitives: PieChart's Layer wrapper crashes when rendered on the server. -->
			<LayerChart ssr width={size} height={size} data={rows} x="count">
				<Svg>
					<Group center>
						<Pie sort={null} {innerRadius} padAngle={0.01}>
							{#snippet children({ arcs })}
								{#each arcs as arc (arc.data.platform)}
									<Arc
										startAngle={arc.startAngle}
										endAngle={arc.endAngle}
										padAngle={arc.padAngle}
										{innerRadius}
										fill={arc.data.color}
									/>
								{/each}
							{/snippet}
						</Pie>
					</Group>
				</Svg>
			</LayerChart>
		{:else}
			<div class="absolute inset-0 rounded-full border-[34px] border-secondary"></div>
		{/if}
		<div
			class="pointer-events-none absolute inset-0 grid place-items-center text-center"
			aria-hidden="true"
		>
			<div>
				<div class="font-mono text-[1.375rem] leading-none font-semibold">
					{counts ? formatNumber(total) : '…'}
				</div>
				<div class="mt-1 text-[0.6875rem] font-medium text-muted-foreground">
					{m.monitor_new_posts_unit()}
				</div>
			</div>
		</div>
	</Chart.Container>

	<table class="min-w-40 flex-1 text-[0.8125rem]">
		<caption class="sr-only">{m.monitor_posts_description()}</caption>
		<thead class="sr-only">
			<tr>
				<th scope="col">{m.overview_creators_platform()}</th>
				<th scope="col">{m.monitor_new_posts_unit()}</th>
				<th scope="col">{m.overview_col_share()}</th>
			</tr>
		</thead>
		<tbody>
			{#each rows as row (row.platform)}
				<tr class="border-t border-rule">
					<th scope="row" class="py-2 text-left font-medium">
						<span class="flex items-center gap-2.5">
							<span class="size-2.5 shrink-0 rounded-[2px]" style:background={row.color}></span>
							{config[row.platform].label}
						</span>
					</th>
					<td class="py-2 text-right font-mono text-xs text-muted-foreground">
						{#if counts}{formatNumber(row.count)}{:else}{@render pending()}{/if}
					</td>
					<td class="w-12 py-2 text-right font-mono font-semibold">
						{#if counts}{row.share}%{:else}{@render pending()}{/if}
					</td>
				</tr>
			{/each}
		</tbody>
		<!-- The donut's centre shows the total; the table says it too. -->
		{#if counts}
			<tfoot class="sr-only">
				<tr>
					<th scope="row">{m.monitor_total()}</th>
					<td>{formatNumber(total)}</td>
					<td>100%</td>
				</tr>
			</tfoot>
		{/if}
	</table>
</div>
