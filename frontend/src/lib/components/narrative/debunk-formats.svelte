<script lang="ts">
	import type { DebunkFormat } from '#lib/domain/narrative-detail.js';
	import { cn } from '#lib/utils.js';
	import { MESSAGE_FORMAT_LABEL } from '../common/labels';

	interface Props {
		/** Suggested formats, most reach first. */
		formats: readonly DebunkFormat[];
		emptyText?: string;
		class?: string;
	}

	let { formats, emptyText = 'No suggestions yet.', class: className }: Props = $props();

	/** The design names audio debunks "Audio", not "Voice note". */
	function formatName(item: DebunkFormat) {
		const format = item.format === 'audio' ? 'Audio' : MESSAGE_FORMAT_LABEL[item.format];
		return `${format} · ${item.language}`;
	}
</script>

{#if formats.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<ul class={className}>
		{#each formats as item (item.id)}
			<li class="flex items-center gap-2.5 border-t border-rule py-2.5 first:border-t-0">
				<div class="min-w-0 flex-1">
					<div class="text-[0.84375rem] font-medium">{formatName(item)}</div>
					<div class="font-mono text-[0.71875rem] text-faint-foreground">
						{item.reachPct}% of messages
					</div>
				</div>
				<span
					data-available={item.available}
					class="rounded-full px-2.5 py-[0.1875rem] text-[0.71875rem] font-medium whitespace-nowrap data-[available=false]:bg-verdict-false data-[available=false]:text-verdict-false-foreground data-[available=true]:bg-verdict-misleading data-[available=true]:text-verdict-misleading-foreground"
				>
					{item.available ? 'Available' : 'Missing'}
				</span>
			</li>
		{/each}
	</ul>
{/if}
