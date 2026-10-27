<script lang="ts">
	import PlayIcon from '@lucide/svelte/icons/play';
	import {
		costPerRun,
		isRunning,
		latestFinishedRun,
		selectRun,
		volumeTotals,
		type MonitorSnapshot
	} from '#lib/domain/monitor.js';
	import { TIME_RANGES, type TimeRange } from '#lib/domain/time-range.js';
	import { m } from '#lib/paraglide/messages.js';
	import { optionalEnhance, type FormEnhance } from '#lib/utils/enhance.js';
	import { formatLogTime, formatNumber, formatUsd } from '#lib/utils/format.js';
	import KpiGroup from '../common/kpi-group.svelte';
	import KpiTile from '../common/kpi-tile.svelte';
	import { RUN_TYPE_LABEL, TIME_RANGE_LABEL } from '../common/labels';
	import PageHeader from '../common/page-header.svelte';
	import SectionCard from '../common/section-card.svelte';
	import SegmentedLinks from '../common/segmented-links.svelte';
	import { Button } from '../ui/button/index.js';
	import ProcessingVolumeChart from './processing-volume-chart.svelte';
	import RunPlatformBreakdown from './run-platform-breakdown.svelte';
	import RunStatusBreakdown from './run-status-breakdown.svelte';
	import RunTable from './run-table.svelte';

	interface Props {
		snapshot: MonitorSnapshot;
		range: TimeRange;
		/** The run to inspect; the newest one when absent or unknown. */
		selectedRunId?: string | null;
		/** This page with another time range, e.g. "?range=7d". */
		rangeHref: (range: TimeRange) => string;
		/** This page with another run selected. */
		runHref?: (id: string) => string;
		/** Where "Run now" posts to. */
		runAction?: string;
		/** Why starting a run failed, from the action. */
		runError?: string;
		/** Upgrades the "Run now" form, e.g. SvelteKit's `enhance`. */
		enhance?: FormEnhance;
	}

	let {
		snapshot,
		range,
		selectedRunId,
		rangeHref,
		runHref,
		runAction = '?/run',
		runError,
		enhance
	}: Props = $props();

	const windowLabel = $derived(TIME_RANGE_LABEL[range]);
	const now = $derived(snapshot.updatedAt);
	const minutes = (value: number) => m.monitor_minutes({ minutes: formatNumber(value) });

	const totals = $derived(volumeTotals(snapshot.volume.buckets));
	const latest = $derived(latestFinishedRun(snapshot.runs)?.result);
	const running = $derived(snapshot.runs.some(isRunning));
	const selected = $derived(selectRun(snapshot.runs, selectedRunId));
	const selectedTime = $derived(selected ? formatLogTime(selected.startedAt, now) : '');
</script>

<main class="mx-auto flex max-w-[85rem] flex-col gap-5 p-[clamp(1rem,3vw,2rem)]">
	<PageHeader
		title={m.nav_monitor()}
		eyebrow={m.monitor_eyebrow({ window: windowLabel })}
		class="mb-0.5"
	>
		{#snippet actions()}
			<SegmentedLinks
				label={m.monitor_time_range()}
				current={range}
				options={TIME_RANGES.map((value) => ({
					value,
					label: TIME_RANGE_LABEL[value],
					href: rangeHref(value)
				}))}
			/>
			<!-- A plain POST form: works without JavaScript; `enhance` upgrades it. -->
			<form method="POST" action={runAction} use:optionalEnhance={enhance}>
				<!--
					aria-disabled rather than disabled, so the button stays in the tab
					order and screen readers can still find it and hear "Running…".
				-->
				<Button
					type="submit"
					aria-disabled={running || undefined}
					aria-describedby={runError ? 'run-error' : undefined}
					class="aria-disabled:cursor-wait aria-disabled:opacity-50"
					onclick={(event) => {
						if (running) event.preventDefault();
					}}
				>
					<PlayIcon aria-hidden="true" />
					{running ? m.monitor_running() : m.monitor_run_now()}
				</Button>
			</form>
		{/snippet}
	</PageHeader>

	{#if runError}
		<p id="run-error" role="alert" class="-mt-2 text-[0.8125rem] font-medium text-destructive">
			{runError}
		</p>
	{/if}

	<KpiGroup label={m.monitor_kpi_group({ window: windowLabel })}>
		<KpiTile
			label={m.monitor_kpi_volume()}
			value={formatNumber(totals.processed)}
			caption={m.monitor_kpi_volume_note({
				failed: formatNumber(totals.failed),
				window: windowLabel
			})}
		/>
		<KpiTile
			label={m.monitor_kpi_discovered()}
			value={latest ? formatNumber(latest.discovered) : '—'}
			caption={latest
				? m.monitor_kpi_discovered_note({ count: formatNumber(latest.newPosts) })
				: undefined}
		/>
		<KpiTile
			label={m.monitor_kpi_duration()}
			value={minutes(snapshot.avgDurationMinutes)}
			caption={latest
				? m.monitor_kpi_duration_note({ duration: minutes(latest.durationMinutes) })
				: undefined}
		/>
		<KpiTile
			label={m.monitor_kpi_cost()}
			value={formatUsd(snapshot.costUsd)}
			caption={m.monitor_kpi_cost_note({ cost: formatUsd(costPerRun(snapshot)) })}
		/>
	</KpiGroup>

	<div class="flex flex-wrap items-start gap-5">
		<SectionCard
			title={m.monitor_runs_title()}
			description={m.monitor_runs_description()}
			flush
			class="flex-[2_1_38.75rem]"
		>
			{#snippet actions()}
				<span class="font-mono text-xs font-medium text-muted-foreground">
					{m.monitor_runs_count({
						shown: formatNumber(snapshot.runs.length),
						total: formatNumber(Math.max(snapshot.runCount, snapshot.runs.length))
					})}
				</span>
			{/snippet}
			<RunTable runs={snapshot.runs} selectedId={selected?.id} {runHref} {now} />
			<!-- Picking a run keeps focus on its link; this says what changed below. -->
			<p class="sr-only" aria-live="polite">
				{selected ? m.monitor_selected_run({ time: selectedTime }) : ''}
			</p>
		</SectionCard>

		{#if selected}
			<SectionCard
				title={m.monitor_status_title()}
				eyebrow={selected.executedBy
					? m.monitor_status_kicker_manual({
							time: selectedTime,
							type: RUN_TYPE_LABEL[selected.type],
							name: selected.executedBy
						})
					: m.monitor_status_kicker({ time: selectedTime, type: RUN_TYPE_LABEL[selected.type] })}
				class="flex-[1_1_20rem]"
			>
				<RunStatusBreakdown run={selected} />
			</SectionCard>

			<SectionCard
				title={m.monitor_posts_title({ time: selectedTime })}
				description={m.monitor_posts_description()}
				class="flex-[1_1_22.5rem]"
			>
				<RunPlatformBreakdown counts={selected.result?.platformCounts ?? null} />
			</SectionCard>
		{/if}

		<SectionCard
			title={m.monitor_volume_title()}
			description={snapshot.volume.bucketHours >= 24
				? m.monitor_volume_description_day()
				: m.monitor_volume_description_window({ hours: formatNumber(snapshot.volume.bucketHours) })}
			class="flex-[2_1_35rem]"
		>
			{#snippet actions()}
				<div class="text-right">
					<div class="font-mono text-xl font-semibold">{formatNumber(totals.processed)}</div>
					<div class="text-[0.6875rem] font-medium text-muted-foreground">
						{m.monitor_volume_total_label({ window: windowLabel })}
					</div>
				</div>
			{/snippet}
			<ProcessingVolumeChart
				buckets={snapshot.volume.buckets}
				bucketHours={snapshot.volume.bucketHours}
			/>
		</SectionCard>

		<p class="flex-[1_1_100%] text-xs/normal text-pretty text-muted-foreground">
			<span aria-hidden="true">ⓘ</span>
			{m.monitor_definition()}
		</p>
	</div>
</main>
