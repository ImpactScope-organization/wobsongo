<script lang="ts">
	import { isSpiking, type NarrativeSummary } from '#lib/domain/narrative.js';
	import { formatNumber, formatPercent } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import HarmScore from '../common/harm-score.svelte';
	import ReviewStatusBadge from '../common/review-status-badge.svelte';
	import Sparkline from '../common/sparkline.svelte';
	import VerdictBadge from '../common/verdict-badge.svelte';

	interface Props {
		/** Already ranked, highest volume first. */
		narratives: readonly NarrativeSummary[];
		/** Accessible name for the list. */
		label?: string;
		/** Link for a narrative's detail page. */
		hrefFor?: (id: string) => string;
		emptyText?: string;
		class?: string;
	}

	let {
		narratives,
		label = 'Narratives',
		hrefFor = (id) => `/narratives/${encodeURIComponent(id)}`,
		emptyText = 'No narratives in this window.',
		class: className
	}: Props = $props();
</script>

{#if narratives.length === 0}
	<p
		class={cn(
			'border-t border-rule px-5 py-8 text-center text-sm text-muted-foreground',
			className
		)}
	>
		{emptyText}
	</p>
{:else}
	<ol aria-label={label} class={className}>
		{#each narratives as narrative, index (narrative.id)}
			{@const spiking = isSpiking(narrative.growthPct)}
			<li class="relative border-t border-rule hover:bg-muted has-focus-visible:bg-muted">
				<div class="flex flex-wrap items-center gap-x-[1.125rem] gap-y-3.5 px-5 py-3.5">
					<span class="w-[1.125rem] font-mono text-[0.8125rem] font-semibold text-faint-foreground">
						{index + 1}
					</span>

					<div class="flex min-w-0 flex-[1_1_16rem] flex-col gap-1.5">
						<!-- The title link's ::after stretches over the whole row. -->
						<a
							href={hrefFor(narrative.id)}
							class="text-[0.9375rem]/snug font-semibold text-pretty outline-none after:absolute after:inset-0 focus-visible:underline"
						>
							{narrative.title}
						</a>
						<div
							class="flex flex-wrap items-center gap-1.5 text-[0.71875rem] text-muted-foreground"
						>
							<VerdictBadge verdict={narrative.verdict} />
							<span>{narrative.domain}</span>
							<span aria-hidden="true">·</span>
							<HarmScore harm={narrative.harm} />
						</div>
					</div>

					<div class="ml-auto flex items-center gap-[1.125rem]">
						<Sparkline
							values={narrative.trend}
							tone={spiking ? 'signal' : 'muted'}
							label="Volume trend{spiking ? ', spiking' : ''}"
						/>
						<div class="w-16 text-right font-mono">
							<div class="text-[0.9375rem] font-semibold">
								{formatNumber(narrative.volume)}<span class="sr-only">&nbsp;messages</span>
							</div>
							<div
								data-spiking={spiking || undefined}
								class="text-[0.71875rem] font-medium text-muted-foreground data-spiking:text-signal"
							>
								{formatPercent(narrative.growthPct, { signed: true })}<span class="sr-only"
									>&nbsp;growth</span
								>
							</div>
						</div>
						<ReviewStatusBadge status={narrative.status} class="w-28" />
					</div>
				</div>
			</li>
		{/each}
	</ol>
{/if}
