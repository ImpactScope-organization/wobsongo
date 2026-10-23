<script lang="ts">
	import { isHighFollowUp, type DebunkOutcome } from '#lib/domain/operations.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatNumber } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** Already sorted for display. */
		outcomes: readonly DebunkOutcome[];
		emptyText?: string;
		class?: string;
	}

	let { outcomes, emptyText = m.operations_debunks_empty(), class: className }: Props = $props();

	const HEAD = 'py-1.5 text-left text-[0.6875rem] font-medium text-faint-foreground';
</script>

<!-- Colour alone can't flag a high rate, so screen readers get it in words. -->
{#snippet meter(value: number, high: boolean)}
	<div class="flex items-center gap-2">
		<div class="h-1.5 flex-1 overflow-hidden rounded-[3px] bg-rule" aria-hidden="true">
			<div
				class={cn('h-full', high ? 'bg-signal' : 'bg-platform-whatsapp')}
				style:width="{Math.min(Math.max(value, 0), 100)}%"
			></div>
		</div>
		<span class={cn('w-9 font-mono text-xs font-semibold', high && 'text-signal')}
			>{value}%{#if high}<span class="sr-only"
					><!-- Leading NBSP: Svelte trims ordinary leading whitespace in the span. -->&nbsp;{m.operations_high_note()}</span
				>{/if}</span
		>
	</div>
{/snippet}

{#if outcomes.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<div class={cn('overflow-x-auto', className)}>
		<table class="w-full min-w-[36rem]">
			<caption class="sr-only">{m.operations_debunk_caption()}</caption>
			<thead>
				<tr>
					<th scope="col" class={HEAD}>{m.operations_col_debunk()}</th>
					<th scope="col" class={cn(HEAD, 'w-20 pr-4 text-right')}>{m.operations_col_sent()}</th>
					<th scope="col" class={cn(HEAD, 'w-44 pr-4')}>{m.operations_col_helpful()}</th>
					<th scope="col" class={cn(HEAD, 'w-44')}>{m.operations_col_asked_again()}</th>
				</tr>
			</thead>
			<tbody>
				{#each outcomes as outcome (outcome.id)}
					<tr class="border-t border-rule">
						<th scope="row" class="py-2.5 pr-4 text-left font-normal">
							<span class="block text-[0.84375rem] font-medium">{outcome.title}</span>
							<span class="block font-mono text-[0.71875rem] text-faint-foreground">
								{outcome.formats}
							</span>
						</th>
						<td class="pr-4 text-right font-mono text-[0.8125rem] font-medium">
							{formatNumber(outcome.sent)}
						</td>
						<td class="pr-4">{@render meter(outcome.helpfulPct, false)}</td>
						<td data-high={isHighFollowUp(outcome) || undefined}>
							{@render meter(outcome.followUpPct, isHighFollowUp(outcome))}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
