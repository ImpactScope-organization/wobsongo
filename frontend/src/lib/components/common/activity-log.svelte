<script lang="ts">
	import type { ActivityEntry } from '#lib/domain/review.js';
	import { formatClockTime, formatLogTime } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		entries: readonly ActivityEntry[];
		/** Reference time for "Today"/"Yesterday"; pass the injected clock's now. */
		now: Date;
		/** `compact`: clock time and text only, for short same-day logs. */
		variant?: 'full' | 'compact';
		emptyText?: string;
		class?: string;
	}

	let {
		entries,
		now,
		variant = 'full',
		emptyText = 'No activity yet.',
		class: className
	}: Props = $props();
</script>

{#if entries.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<ol class={cn('flex flex-col', className)}>
		{#each entries as entry (entry.id)}
			<li
				data-variant={variant}
				class="flex flex-wrap gap-x-3 gap-y-0.5 border-t border-rule py-2.5 text-[0.8125rem] first:border-t-0 data-[variant=compact]:flex-nowrap data-[variant=compact]:gap-x-2.5 data-[variant=compact]:py-[0.4375rem] data-[variant=compact]:text-[0.78125rem]/snug"
			>
				{#if variant === 'compact'}
					<time
						datetime={entry.at.toISOString()}
						title={formatLogTime(entry.at, now)}
						class="w-11 shrink-0 pt-px font-mono text-[0.6875rem] font-medium text-faint-foreground"
					>
						{formatClockTime(entry.at)}
					</time>
				{:else}
					<time
						datetime={entry.at.toISOString()}
						class="w-28 shrink-0 font-mono text-xs text-faint-foreground"
					>
						{formatLogTime(entry.at, now)}
					</time>
					<span class="w-16 shrink-0 font-medium">{entry.actor ?? 'System'}</span>
				{/if}
				<span class="min-w-0 flex-1 text-pretty">{entry.text}</span>
			</li>
		{/each}
	</ol>
{/if}
