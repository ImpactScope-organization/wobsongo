<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { NarrativeDetail } from '#lib/domain/narrative-detail.js';
	import { cn } from '#lib/utils.js';
	import ReviewStatusBadge from '../common/review-status-badge.svelte';
	import VerdictBadge from '../common/verdict-badge.svelte';

	interface Props {
		narrative: Pick<NarrativeDetail, 'title' | 'summary' | 'verdict' | 'status' | 'domain'>;
		/** Page actions, e.g. the dismiss menu and an edit-reply link. */
		actions?: Snippet;
		class?: string;
	}

	let { narrative, actions, class: className }: Props = $props();
</script>

<header class={cn('flex flex-wrap items-start justify-between gap-x-6 gap-y-4', className)}>
	<div class="min-w-0 flex-[1_1_30rem]">
		<div class="mb-2.5 flex flex-wrap items-center gap-2">
			<VerdictBadge verdict={narrative.verdict} class="px-2 py-1 text-[0.6875rem]" />
			<ReviewStatusBadge status={narrative.status} class="py-[0.1875rem] text-xs" />
			<span class="text-xs font-medium text-muted-foreground">{narrative.domain}</span>
		</div>
		<h1
			class="mb-2 text-[clamp(1.5rem,3.2vw,2.25rem)] leading-[1.1] font-extrabold tracking-[-0.025em] text-pretty"
		>
			{narrative.title}
		</h1>
		<p class="max-w-[45rem] text-sm/normal text-pretty text-muted-foreground">
			{narrative.summary}
		</p>
	</div>
	{#if actions}
		<div class="flex flex-none flex-wrap gap-2 whitespace-nowrap">{@render actions()}</div>
	{/if}
</header>
