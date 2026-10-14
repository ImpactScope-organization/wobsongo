<script lang="ts">
	import type { SampleMessage } from '#lib/domain/narrative-detail.js';
	import { cn } from '#lib/utils.js';
	import { PLATFORM_SHORT_LABEL } from '../common/labels';

	interface Props {
		/** Already anonymised on the server: names, numbers and group names removed. */
		samples: readonly SampleMessage[];
		emptyText?: string;
		class?: string;
	}

	let { samples, emptyText = 'No sample messages.', class: className }: Props = $props();
</script>

{#if samples.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<ul class={cn('grid grid-cols-[repeat(auto-fit,minmax(min(100%,15rem),1fr))] gap-3', className)}>
		{#each samples as sample (sample.id)}
			<li class="flex flex-col gap-2.5 rounded-lg border border-rule bg-muted p-3.5">
				<figure class="contents">
					<figcaption
						class="text-eyebrow order-first flex flex-wrap gap-1.5 text-[0.65625rem] font-semibold tracking-[0.04em] text-muted-foreground"
					>
						<span>{PLATFORM_SHORT_LABEL[sample.platform]}</span>
						<span aria-hidden="true">·</span>
						<span>{sample.formatLabel}</span>
						<span aria-hidden="true">·</span>
						<span>{sample.language}</span>
					</figcaption>
					<blockquote class="text-sm/normal text-pretty">“{sample.text}”</blockquote>
				</figure>
				<p class="text-xs/[1.45] text-muted-foreground italic">{sample.note}</p>
			</li>
		{/each}
	</ul>
{/if}
