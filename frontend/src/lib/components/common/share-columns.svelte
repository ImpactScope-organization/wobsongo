<script lang="ts" module>
	import type { StackedSegment } from './stacked-bar.svelte';

	export interface ShareColumn {
		key: string;
		/** Axis label, e.g. "W34". */
		label: string;
		/** Usually the same segment keys in every column; a missing key reads as "—". */
		segments: readonly StackedSegment[];
	}
</script>

<script lang="ts">
	import { percentShares } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** Oldest first. */
		columns: readonly ShareColumn[];
		/** Caption of the equivalent data table, e.g. "Share of claims by platform, weekly". */
		caption: string;
		/** Header of the column-label column, e.g. "Week". */
		columnHeader?: string;
		/** Height of the bars, e.g. "h-52". */
		barsClass?: string;
		emptyText?: string;
		class?: string;
	}

	let {
		columns,
		caption,
		columnHeader = 'Week',
		barsClass = 'h-52',
		emptyText = 'No data in this window.',
		class: className
	}: Props = $props();

	// 100% stacked columns as flex boxes, no JavaScript. The visual is hidden from
	// screen readers; a visually hidden table carries the same numbers.
	const rows = $derived(
		columns.map((column) => ({
			...column,
			shares: percentShares(column.segments.map((segment) => segment.value))
		}))
	);
	// Every segment key across the columns, in first-seen order, so the table's
	// headers and cells line up even if a column lacks a segment.
	const series = $derived([
		...new Map(columns.flatMap((column) => column.segments).map((s) => [s.key, s])).values()
	]);
</script>

{#if columns.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<figure class={className}>
		<div aria-hidden="true" class={cn('flex gap-[clamp(0.25rem,1vw,0.625rem)]', barsClass)}>
			{#each rows as column (column.key)}
				<div class="flex min-w-0 flex-1 flex-col gap-1.5">
					<div class="flex flex-1 flex-col gap-0.5 overflow-hidden rounded bg-secondary">
						{#each column.segments as segment (segment.key)}
							{#if segment.value > 0}
								<div
									data-segment={segment.key}
									class={segment.class}
									style:flex-grow={segment.value}
									style:flex-basis="0"
								></div>
							{/if}
						{/each}
					</div>
					<span
						class="overflow-hidden text-center font-mono text-[0.65625rem] font-medium whitespace-nowrap text-faint-foreground"
					>
						{column.label}
					</span>
				</div>
			{/each}
		</div>

		<table class="sr-only">
			<caption>{caption}</caption>
			<thead>
				<tr>
					<th scope="col">{columnHeader}</th>
					{#each series as segment (segment.key)}
						<th scope="col">{segment.label}</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each rows as column (column.key)}
					<tr>
						<th scope="row">{column.label}</th>
						{#each series as segment (segment.key)}
							{@const at = column.segments.findIndex((own) => own.key === segment.key)}
							<td>{at === -1 ? '—' : `${column.shares[at]}%`}</td>
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</figure>
{/if}
