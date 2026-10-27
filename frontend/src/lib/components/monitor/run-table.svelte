<script lang="ts">
	import type { ProcessingRun } from '#lib/domain/monitor.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatLogTime, formatNumber, formatUsd } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { RUN_TYPE_LABEL } from '../common/labels';

	interface Props {
		/** Newest first. */
		runs: readonly ProcessingRun[];
		/** The run whose details the page shows. */
		selectedId?: string;
		/** This page with another run selected, e.g. "?run=run-202610121400". */
		runHref?: (id: string) => string;
		/** "Now", for "Today 14:00" style run times. */
		now: Date;
		class?: string;
	}

	let {
		runs,
		selectedId,
		runHref = (id) => `?run=${encodeURIComponent(id)}`,
		now,
		class: className
	}: Props = $props();

	const head =
		'px-2.5 py-2 font-mono text-[0.65625rem] font-medium tracking-[0.05em] whitespace-nowrap text-faint-foreground uppercase first:pl-5 last:pr-5';
	const cell = 'px-2.5 py-[0.6875rem] whitespace-nowrap first:pl-5 last:pr-5';
	const number = 'text-right font-mono text-[0.8125rem] font-medium';
</script>

<!-- A running run has no numbers yet: "…" on screen, words for screen readers. -->
{#snippet pending()}
	<span aria-hidden="true">…</span><span class="sr-only">{m.monitor_pending()}</span>
{/snippet}

{#if runs.length === 0}
	<p class={cn('px-5 pb-[1.125rem] text-[0.8125rem] text-muted-foreground', className)}>
		{m.monitor_runs_empty()}
	</p>
{:else}
	<!-- Scrolls sideways on narrow screens rather than squashing the columns. -->
	<div class={cn('overflow-x-auto', className)}>
		<table class="w-full min-w-[43.75rem] border-collapse text-left">
			<caption class="sr-only">{m.monitor_runs_title()}</caption>
			<thead class="border-t border-rule bg-muted">
				<tr>
					<th scope="col" class={head}>{m.monitor_col_run_time()}</th>
					<th scope="col" class={head}>{m.monitor_col_type()}</th>
					<th scope="col" class={head}>{m.monitor_col_executed_by()}</th>
					<th scope="col" class={cn(head, 'text-right')}>{m.monitor_col_discovered()}</th>
					<th scope="col" class={cn(head, 'text-right')}>{m.monitor_col_new()}</th>
					<th scope="col" class={cn(head, 'text-right')}>{m.monitor_col_duration()}</th>
					<th scope="col" class={cn(head, 'text-right')}>{m.monitor_col_failed()}</th>
					<th scope="col" class={cn(head, 'text-right')}>{m.monitor_col_cost()}</th>
				</tr>
			</thead>
			<tbody>
				{#each runs as run (run.id)}
					{@const result = run.result}
					{@const selected = run.id === selectedId}
					<!-- The run-time link stretches over the row, so the whole row selects the run. -->
					<tr
						data-selected={selected || undefined}
						class="relative border-t border-rule transition-colors hover:bg-muted data-selected:bg-background data-selected:shadow-[inset_3px_0_0_var(--foreground)]"
					>
						<th scope="row" class={cn(cell, 'font-mono text-[0.8125rem] font-semibold')}>
							<a
								href={runHref(run.id)}
								aria-current={selected ? 'true' : undefined}
								data-sveltekit-noscroll
								data-sveltekit-keepfocus
								class="outline-none after:absolute after:inset-0 focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
							>
								{formatLogTime(run.startedAt, now)}
							</a>
						</th>
						<td class={cell}>
							<span
								data-type={run.type}
								class="inline-flex rounded-full px-2 py-[0.1875rem] font-mono text-[0.6875rem] font-medium data-[type=automatic]:bg-secondary data-[type=automatic]:text-verdict-unverified-foreground data-[type=manual]:bg-platform-facebook/12 data-[type=manual]:text-platform-facebook"
							>
								{RUN_TYPE_LABEL[run.type]}
							</span>
						</td>
						<td
							class={cn(
								cell,
								'text-[0.78125rem] font-medium',
								!run.executedBy && 'text-faint-foreground'
							)}
						>
							{run.executedBy ?? m.monitor_run_by_system()}
						</td>
						{#if result}
							<td class={cn(cell, number)}>{formatNumber(result.discovered)}</td>
							<td class={cn(cell, number, 'font-semibold')}>{formatNumber(result.newPosts)}</td>
							<td class={cn(cell, number)}>
								{m.monitor_minutes({ minutes: formatNumber(result.durationMinutes) })}
							</td>
							<td
								class={cn(
									cell,
									number,
									result.failed > 0 ? 'text-signal' : 'text-faint-foreground'
								)}
							>
								{formatNumber(result.failed)}
							</td>
							<td class={cn(cell, number)}>{formatUsd(result.costUsd)}</td>
						{:else}
							<td class={cn(cell, number)}>{@render pending()}</td>
							<td class={cn(cell, number)}>{@render pending()}</td>
							<td class={cn(cell, number, 'text-verdict-misleading-foreground')}>
								{m.monitor_running()}
							</td>
							<td class={cn(cell, number, 'text-faint-foreground')}>{@render pending()}</td>
							<td class={cn(cell, number)}>{@render pending()}</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
