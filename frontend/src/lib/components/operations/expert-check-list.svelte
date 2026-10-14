<script lang="ts">
	import type { ExpertCheck } from '#lib/domain/operations.js';
	import type { FormEnhance } from '#lib/utils/enhance.js';
	import { cn } from '#lib/utils.js';
	import { CHECK_REASON_LABEL } from '../common/labels';
	import ReviewActions from '../common/review-actions.svelte';
	import VerdictBadge from '../common/verdict-badge.svelte';

	interface Props {
		/** Most urgent first; the server decides the order. */
		checks: readonly ExpertCheck[];
		/** Form action URL for every item's review form, e.g. "?/review". */
		action: string;
		/** Editor page behind each item's "Override…". */
		overrideHrefFor: (checkId: string) => string;
		enhance?: FormEnhance;
		emptyText?: string;
		class?: string;
	}

	let {
		checks,
		action,
		overrideHrefFor,
		enhance,
		emptyText = 'Nothing waiting for an expert.',
		class: className
	}: Props = $props();

	/** "Harm 4 · spiking", "Low confidence · Mooré audio": sentence case, note kept as is. */
	function flag(check: ExpertCheck): string {
		const reasons = [
			...(check.harm === undefined ? [] : [`Harm ${check.harm}`]),
			...check.reasons.map((reason) => CHECK_REASON_LABEL[reason])
		].map((label, index) => (index === 0 ? label : label.toLowerCase()));
		return [...reasons, ...(check.note ? [check.note] : [])].join(' · ');
	}
</script>

{#if checks.length === 0}
	<p class={cn('px-5 pb-5 text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<ol aria-label="Expert review queue" class={className}>
		{#each checks as check (check.id)}
			{@const decision = check.review.kind === 'decided' ? check.review.decision : undefined}
			<li
				data-decision={decision}
				class="flex flex-wrap items-start gap-x-5 gap-y-3 border-t border-rule px-5 py-3.5 transition-opacity data-[decision=dismiss]:opacity-55"
			>
				<div class="flex min-w-0 flex-[1_1_22.5rem] flex-col gap-[0.4375rem]">
					<div class="flex flex-wrap items-center gap-2">
						<VerdictBadge verdict={check.engine.verdict} />
						<span class="font-mono text-[0.71875rem] font-medium text-muted-foreground">
							conf. {check.engine.confidence.toFixed(2)}
						</span>
						<span
							class="rounded-full border border-signal-soft px-2 py-px text-[0.6875rem] font-medium text-signal"
						>
							<span class="sr-only">Flagged for:</span>
							{flag(check)}
						</span>
					</div>
					<h3 class="font-sans text-[0.90625rem]/[1.3] font-semibold text-pretty">{check.title}</h3>
					<p
						class="rounded-md bg-background px-[0.6875rem] py-[0.5625rem] text-[0.8125rem]/[1.45] text-pretty text-verdict-unverified-foreground"
					>
						<span
							class="mr-1.5 font-mono text-[0.65625rem] font-semibold tracking-[0.04em] text-muted-foreground uppercase"
						>
							Engine reply
						</span>
						{check.engine.explanation}
					</p>
				</div>
				<ReviewActions
					itemId={check.id}
					itemLabel={check.title}
					state={check.review}
					{action}
					overrideHref={overrideHrefFor(check.id)}
					{enhance}
					class="ml-auto pt-0.5"
				/>
			</li>
		{/each}
	</ol>
{/if}
