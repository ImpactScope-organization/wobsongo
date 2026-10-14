<script lang="ts">
	import type { EngineVerdict } from '#lib/domain/narrative-detail.js';
	import type { ActivityEntry, ReviewState } from '#lib/domain/review.js';
	import type { FormEnhance } from '#lib/utils/enhance.js';
	import { cn } from '#lib/utils.js';
	import ActivityLog from '../common/activity-log.svelte';
	import { VERDICT_LABEL } from '../common/labels';
	import ReviewActions from '../common/review-actions.svelte';

	interface Props {
		narrativeId: string;
		/** Narrative title, naming the review form for screen readers. */
		narrativeTitle: string;
		engine: EngineVerdict;
		state: ReviewState;
		/** Form action URL, e.g. "?/review". Posts `id` and `decision` (approve or undo). */
		action: string;
		/** Reply editor page behind "Correct…", e.g. "/narratives/N-0412/reply". */
		overrideHref: string;
		log: readonly ActivityEntry[];
		/** Pass the injected clock's now. */
		now: Date;
		enhance?: FormEnhance;
		class?: string;
	}

	let {
		narrativeId,
		narrativeTitle,
		engine,
		state,
		action,
		overrideHref,
		log,
		now,
		enhance,
		class: className
	}: Props = $props();

	const VERDICT_TONE = {
		false: 'text-verdict-false-strong',
		misleading: 'text-verdict-misleading-foreground',
		unverified: 'text-muted-foreground'
	} as const;
</script>

<div class={cn('flex flex-col gap-3', className)}>
	<div class="rounded-lg border border-rule bg-muted px-3.5 py-3">
		<div
			class="mb-1.5 flex justify-between gap-2 font-mono text-[0.6875rem] font-semibold uppercase"
		>
			<span class="tracking-[0.05em] text-muted-foreground">API verdict</span>
			<span class={VERDICT_TONE[engine.verdict]}>
				{VERDICT_LABEL[engine.verdict]} · {engine.confidence.toFixed(2)}<span class="sr-only"
					>&nbsp;confidence</span
				>
			</span>
		</div>
		<p class="text-[0.8125rem]/[1.45] text-pretty">{engine.explanation}</p>
	</div>

	<ReviewActions
		itemId={narrativeId}
		itemLabel={narrativeTitle}
		{state}
		{action}
		{enhance}
		{overrideHref}
		decisions={['approve', 'override']}
		labels={{ approve: 'Confirm verdict', override: 'Correct…' }}
		fill
		class="mb-1"
	/>

	<section aria-label="Review log">
		<h3 class="text-eyebrow mb-2 font-semibold tracking-[0.05em] text-muted-foreground">
			Review log
		</h3>
		<ActivityLog entries={log} {now} variant="compact" emptyText="No reviews yet." />
	</section>
</div>
