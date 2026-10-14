<script lang="ts">
	import { MESSAGE_INTENTS, type DailyIntents, type MessageIntent } from '#lib/domain/spread.js';
	import { formatWeekday } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { MESSAGE_INTENT_LABEL } from '../common/labels';
	import StackedBar from '../common/stacked-bar.svelte';
	import SwatchLegend from '../common/swatch-legend.svelte';

	interface Props {
		/** Oldest first. */
		days: readonly DailyIntents[];
		emptyText?: string;
		class?: string;
	}

	let { days, emptyText = 'No messages in this window.', class: className }: Props = $props();

	const TONE: Record<MessageIntent, { bar: string; swatch: string }> = {
		verification: { bar: 'bg-primary text-primary-foreground', swatch: 'bg-primary' },
		general: { bar: 'bg-platform-whatsapp text-white', swatch: 'bg-platform-whatsapp' },
		'out-of-scope': { bar: 'bg-faint-foreground text-white', swatch: 'bg-faint-foreground' }
	};
</script>

{#if days.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<div class={className}>
		<dl class="flex flex-col gap-2">
			{#each days as day (day.day.getTime())}
				<div class="grid grid-cols-[2.75rem_minmax(0,1fr)] items-center gap-2.5">
					<dt class="font-mono text-[0.6875rem] font-medium text-faint-foreground">
						<time datetime={day.day.toISOString().slice(0, 10)}>{formatWeekday(day.day)}</time>
					</dt>
					<dd>
						<StackedBar
							size="md"
							describe="percent"
							showPercent
							segments={MESSAGE_INTENTS.map((intent) => ({
								key: intent,
								label: MESSAGE_INTENT_LABEL[intent].label,
								value: day.counts[intent],
								class: TONE[intent].bar
							}))}
						/>
					</dd>
				</div>
			{/each}
		</dl>
		<SwatchLegend
			decorative={false}
			class="mt-4 grid grid-cols-[repeat(auto-fit,minmax(min(100%,11.25rem),1fr))] gap-3.5 border-t border-rule pt-3.5"
			items={MESSAGE_INTENTS.map((intent) => ({
				key: intent,
				...MESSAGE_INTENT_LABEL[intent],
				class: TONE[intent].swatch
			}))}
		/>
	</div>
{/if}
