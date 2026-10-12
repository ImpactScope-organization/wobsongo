<script lang="ts" module>
	import type { ReviewDecision } from '#lib/domain/review.js';

	const DECISION_LABEL: Record<ReviewDecision, string> = {
		approve: 'Approve',
		override: 'Override…',
		dismiss: 'Dismiss',
		'add-fact-check': 'Add fact-check'
	};
</script>

<script lang="ts">
	import type { ReviewState } from '#lib/domain/review.js';
	import { optionalEnhance, type FormEnhance } from '#lib/utils/enhance.js';
	import { cn } from '#lib/utils.js';
	import { REVIEW_OUTCOME_LABEL } from './labels';

	interface Props {
		/** Id of the reviewed item, posted as `id`. */
		itemId: string;
		/** What is being reviewed, for screen readers, e.g. the claim title. */
		itemLabel: string;
		state: ReviewState;
		/** Form action URL, e.g. "?/review". Posts `id` and `decision`. */
		action: string;
		/** Decisions offered while pending; the first is the primary button. */
		decisions?: readonly ReviewDecision[];
		/** Progressive enhancement, e.g. SvelteKit's `enhance`. */
		enhance?: FormEnhance;
		class?: string;
	}

	let {
		itemId,
		itemLabel,
		state,
		action,
		decisions = ['approve', 'override', 'dismiss'],
		enhance,
		class: className
	}: Props = $props();

	const BUTTON =
		'rounded-md px-3 py-1.5 text-xs font-semibold transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50';
</script>

<!-- A plain POST form: works without JavaScript; `enhance` upgrades it. -->
<form method="POST" {action} use:optionalEnhance={enhance} class={cn('flex', className)}>
	<input type="hidden" name="id" value={itemId} />
	<fieldset class="flex flex-wrap items-center gap-1.5">
		<legend class="sr-only">Review “{itemLabel}”</legend>
		{#if state.kind === 'pending'}
			{#each decisions as decision, index (decision)}
				<button
					type="submit"
					name="decision"
					value={decision}
					class={cn(
						BUTTON,
						index === 0
							? 'bg-primary text-primary-foreground hover:bg-primary/85'
							: decision === 'dismiss'
								? 'text-muted-foreground hover:bg-muted hover:text-foreground'
								: 'border bg-card hover:border-foreground'
					)}
				>
					{DECISION_LABEL[decision]}
				</button>
			{/each}
		{:else}
			<span
				role="status"
				data-decision={state.decision}
				class="text-xs font-semibold text-status-verified-foreground data-[decision=dismiss]:text-muted-foreground data-[decision=override]:text-verdict-misleading-foreground"
			>
				{REVIEW_OUTCOME_LABEL[state.decision]}
			</span>
			<button
				type="submit"
				name="decision"
				value="undo"
				class={cn(BUTTON, 'text-muted-foreground underline-offset-2 hover:underline')}
			>
				Undo
			</button>
		{/if}
	</fieldset>
</form>
