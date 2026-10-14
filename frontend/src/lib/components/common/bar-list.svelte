<script lang="ts" module>
	export interface BarListItem {
		key: string;
		label: string;
		value: number;
	}
</script>

<script lang="ts">
	import { formatNumber, percentShares } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		items: readonly BarListItem[];
		/** Table caption for screen readers, e.g. "Messages by language". */
		caption: string;
		/** Header of the label column, for screen readers. */
		labelHeader?: string;
		/** Show each item's share of the total, or its raw value. */
		show?: 'percent' | 'value';
		emptyText?: string;
		class?: string;
	}

	let {
		items,
		caption,
		labelHeader = 'Item',
		show = 'percent',
		emptyText = 'No data in this window.',
		class: className
	}: Props = $props();

	const shares = $derived(percentShares(items.map((item) => item.value)));
	const max = $derived(Math.max(0, ...items.map((item) => item.value)));
</script>

{#if items.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<!-- A table with inline bars: readable as data without the visuals. -->
	<table class={cn('w-full text-[0.8125rem]', className)}>
		<caption class="sr-only">{caption}</caption>
		<thead class="sr-only">
			<tr>
				<th scope="col">{labelHeader}</th>
				<th scope="col">{show === 'percent' ? 'Share' : 'Count'}</th>
			</tr>
		</thead>
		<tbody>
			{#each items as item, index (item.key)}
				<tr>
					<th scope="row" class="w-[4.5rem] py-[0.3125rem] pr-2.5 text-left font-medium"
						>{item.label}</th
					>
					<td class="py-[0.3125rem]">
						<div class="flex items-center gap-2.5">
							<div class="h-2 flex-1 overflow-hidden rounded-[2px] bg-rule" aria-hidden="true">
								<div
									class="h-full bg-primary"
									style:width="{max > 0 ? (item.value / max) * 100 : 0}%"
								></div>
							</div>
							<span class="w-10 text-right font-mono text-xs font-medium">
								{show === 'percent' ? `${shares[index]}%` : formatNumber(item.value)}
							</span>
						</div>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}
