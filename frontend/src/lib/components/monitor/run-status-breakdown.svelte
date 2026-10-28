<script lang="ts">
	import {
		PROCESSING_STATUSES,
		hasPendingPosts,
		isFinalStatus,
		type ProcessingRun
	} from '#lib/domain/monitor.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatNumber, formatUsd } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { PROCESSING_STATUS_DESCRIPTION, PROCESSING_STATUS_LABEL } from '../common/labels';
	import StackedBar from '../common/stacked-bar.svelte';
	import { PROCESSING_STATUS_TONE } from '../common/tones';

	interface Props {
		run: ProcessingRun;
		class?: string;
	}

	let { run, class: className }: Props = $props();

	const result = $derived(run.result);
	const statuses = $derived(
		PROCESSING_STATUSES.map((status) => ({
			status,
			count: result?.statusCounts[status] ?? 0,
			final: isFinalStatus(status)
		}))
	);
	const summary = $derived.by(() => {
		if (!result) return m.monitor_status_running();
		const count = formatNumber(result.newPosts);
		return hasPendingPosts(result)
			? m.monitor_status_new_posts_pending({ count })
			: m.monitor_status_new_posts({ count });
	});
</script>

{#snippet pending()}
	<span aria-hidden="true">…</span><span class="sr-only">{m.monitor_pending()}</span>
{/snippet}

<div class={cn('flex flex-col', className)}>
	<p class="mb-3.5 text-[0.8125rem] text-muted-foreground">{summary}</p>

	<StackedBar
		size="md"
		class="mb-3.5"
		segments={statuses.map(({ status, count }) => ({
			key: status,
			label: PROCESSING_STATUS_LABEL[status],
			value: count,
			class: PROCESSING_STATUS_TONE[status]
		}))}
	/>

	<ul aria-label={m.monitor_status_title()}>
		{#each statuses as { status, count, final } (status)}
			<li
				data-status={status}
				data-empty={count === 0 || undefined}
				class="flex items-start gap-2.5 border-t border-rule py-[0.4375rem] data-empty:opacity-45"
			>
				<span
					class={cn('mt-1 size-2.5 shrink-0 rounded-[2px]', PROCESSING_STATUS_TONE[status])}
					aria-hidden="true"
				></span>
				<div class="min-w-0 flex-1">
					<div class="flex flex-wrap items-baseline gap-2">
						<span class="font-mono text-[0.78125rem] font-semibold">
							{PROCESSING_STATUS_LABEL[status]}
						</span>
						<span
							data-final={final || undefined}
							class="text-[0.65625rem] font-medium text-verdict-misleading-foreground data-final:text-faint-foreground"
						>
							{final ? m.monitor_status_kind_final() : m.monitor_status_kind_progress()}
						</span>
					</div>
					<p class="text-xs/[1.4] text-pretty text-muted-foreground">
						{PROCESSING_STATUS_DESCRIPTION[status]}
					</p>
				</div>
				<span class="shrink-0 font-mono text-[0.8125rem] font-semibold">
					{formatNumber(count)}<span class="sr-only">&nbsp;{m.monitor_new_posts_unit()}</span>
				</span>
			</li>
		{/each}
	</ul>

	<dl class="mt-3 grid grid-cols-2 gap-2.5 border-t pt-3">
		<div>
			<dt class="text-[0.6875rem] font-medium text-muted-foreground">
				{m.monitor_col_processing_time()}
			</dt>
			<dd class="font-mono text-[0.9375rem] font-semibold">
				{#if result}
					{m.monitor_minutes({ minutes: formatNumber(result.durationMinutes) })}
				{:else}
					{@render pending()}
				{/if}
			</dd>
		</div>
		<div>
			<dt class="text-[0.6875rem] font-medium text-muted-foreground">{m.monitor_col_cost()}</dt>
			<dd class="font-mono text-[0.9375rem] font-semibold">
				{#if result}
					{formatUsd(result.costUsd)}
				{:else}
					{@render pending()}
				{/if}
			</dd>
		</div>
	</dl>
</div>
