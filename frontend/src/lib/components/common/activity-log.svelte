<script lang="ts">
	import type { ActivityEntry } from '#lib/domain/review.js';
	import { formatLogTime } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		entries: readonly ActivityEntry[];
		/** Reference time for "Today"/"Yesterday"; pass the injected clock's now. */
		now: Date;
		emptyText?: string;
		class?: string;
	}

	let { entries, now, emptyText = 'No activity yet.', class: className }: Props = $props();
</script>

{#if entries.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<ol class={cn('flex flex-col', className)}>
		{#each entries as entry (entry.id)}
			<li
				class="flex flex-wrap gap-x-3 gap-y-0.5 border-t border-rule py-2.5 text-[0.8125rem] first:border-t-0"
			>
				<time
					datetime={entry.at.toISOString()}
					class="w-28 shrink-0 font-mono text-xs text-faint-foreground"
				>
					{formatLogTime(entry.at, now)}
				</time>
				<span class="w-16 shrink-0 font-medium">{entry.actor ?? 'System'}</span>
				<span class="min-w-48 flex-1 text-pretty">{entry.text}</span>
			</li>
		{/each}
	</ol>
{/if}
