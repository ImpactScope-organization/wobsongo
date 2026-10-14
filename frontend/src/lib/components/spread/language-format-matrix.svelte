<script lang="ts">
	import { MESSAGE_FORMATS } from '#lib/domain/narrative-detail.js';
	import type { LanguageFormatMix } from '#lib/domain/spread.js';
	import { percentShares } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { MESSAGE_FORMAT_LABEL } from '../common/labels';

	interface Props {
		/** Already sorted for display. */
		rows: readonly LanguageFormatMix[];
		emptyText?: string;
		class?: string;
	}

	let { rows, emptyText = 'No messages in this window.', class: className }: Props = $props();

	// Row percentages: how each language splits across formats.
	const table = $derived(
		rows.map((row) => ({
			language: row.language,
			shares: percentShares(MESSAGE_FORMATS.map((format) => row.counts[format]))
		}))
	);
</script>

{#if rows.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<div class={className}>
		<table class="w-full border-separate border-spacing-[3px] text-[0.6875rem]">
			<caption class="sr-only">Share of each language's messages by format</caption>
			<thead>
				<tr>
					<th scope="col" class="w-[4.75rem]"><span class="sr-only">Language</span></th>
					{#each MESSAGE_FORMATS as format (format)}
						<th scope="col" class="font-medium text-muted-foreground">
							{MESSAGE_FORMAT_LABEL[format]}
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each table as row (row.language)}
					<tr>
						<th scope="row" class="text-left text-[0.78125rem] font-medium">{row.language}</th>
						{#each row.shares as share, index (MESSAGE_FORMATS[index])}
							<td
								data-dark={share > 40 || undefined}
								class="h-[2.375rem] rounded text-center font-mono text-xs font-semibold data-dark:text-primary-foreground"
								style:background-color="color-mix(in srgb, var(--primary) {Math.round(
									share * 0.85
								)}%, var(--muted))"
							>
								{share}%
							</td>
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
		<p class="mt-2 text-[0.71875rem] text-faint-foreground">
			Row percentages. Darker = larger share.
		</p>
	</div>
{/if}
