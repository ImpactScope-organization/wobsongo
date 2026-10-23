<script lang="ts">
	import { isFrequentMention, type SubstanceMention } from '#lib/domain/high-harm.js';
	import { formatNumber } from '#lib/utils/format.js';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** Most mentioned first; the server decides the order. */
		substances: readonly SubstanceMention[];
		caption?: string;
		emptyText?: string;
		class?: string;
	}

	let {
		substances,
		caption = m.high_harm_substances_caption(),
		emptyText = m.high_harm_substances_empty(),
		class: className
	}: Props = $props();

	const max = $derived(Math.max(0, ...substances.map((substance) => substance.count)));
</script>

{#if substances.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<!-- A table with inline bars: readable as data without the visuals. -->
	<table class={cn('w-full', className)}>
		<caption class="sr-only">{caption}</caption>
		<thead class="sr-only">
			<tr>
				<th scope="col">{m.high_harm_substance_header()}</th>
				<th scope="col">{m.high_harm_mentions()}</th>
			</tr>
		</thead>
		<tbody>
			{#each substances as substance (substance.id)}
				<tr class="border-t border-rule" data-frequent={isFrequentMention(substance) || undefined}>
					<th scope="row" class="py-2 pr-2.5 text-left font-normal">
						<span class="block text-[0.8125rem] font-medium">{substance.name}</span>
						<span class="block text-[0.71875rem] text-faint-foreground">{substance.practice}</span>
					</th>
					<td class="w-28 py-2">
						<div class="flex items-center gap-2.5">
							<div class="h-1.5 flex-1 overflow-hidden rounded-[3px] bg-rule" aria-hidden="true">
								<div
									class="h-full bg-verdict-misleading-foreground in-data-frequent:bg-signal"
									style:width="{max > 0 ? (substance.count / max) * 100 : 0}%"
								></div>
							</div>
							<span class="w-8 text-right font-mono text-xs font-semibold">
								{formatNumber(substance.count)}
							</span>
						</div>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}
