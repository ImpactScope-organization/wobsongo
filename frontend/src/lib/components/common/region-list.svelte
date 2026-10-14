<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { RegionCount } from '#lib/domain/narrative-detail.js';
	import { cn } from '#lib/utils.js';
	import CountValue from './count-value.svelte';

	interface Props {
		/** Already sorted for display. Small counts arrive suppressed from the server. */
		regions: readonly RegionCount[];
		/** Optional map above the list (none exists yet). */
		map?: Snippet;
		emptyText?: string;
		class?: string;
	}

	let { regions, map, emptyText = 'No location answers yet.', class: className }: Props = $props();

	// Show a topic line on every row once any region has one, so rows align.
	const withTopics = $derived(regions.some((row) => row.topTopic));
</script>

<div class={cn('flex flex-col gap-3', className)}>
	{#if map}{@render map()}{/if}

	{#if regions.length === 0}
		<p class="text-[0.8125rem] text-muted-foreground">{emptyText}</p>
	{:else}
		<table class="w-full text-[0.8125rem]">
			<caption class="sr-only">Messages by region</caption>
			<thead class="sr-only">
				<tr><th scope="col">Region</th><th scope="col">Messages</th></tr>
			</thead>
			<tbody>
				{#each regions as row (row.region)}
					<tr
						data-suppressed={row.count.kind === 'suppressed' || undefined}
						class="border-t border-rule font-medium first:border-t-0 data-suppressed:text-faint-foreground"
					>
						<th scope="row" class="py-[0.4375rem] text-left font-medium">
							{row.region}
							{#if withTopics}
								<span class="block text-[0.71875rem] font-normal text-faint-foreground">
									{row.count.kind === 'suppressed'
										? 'Too few responses to show'
										: (row.topTopic ?? '—')}
								</span>
							{/if}
						</th>
						<td class="py-[0.4375rem] text-right align-baseline text-xs"
							><CountValue count={row.count} /></td
						>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</div>
