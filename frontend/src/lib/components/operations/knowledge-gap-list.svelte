<script lang="ts">
	import type { KnowledgeGap } from '#lib/domain/operations.js';
	import type { FormEnhance } from '#lib/utils/enhance.js';
	import { formatAgo, formatNumber } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import ReviewActions from '../common/review-actions.svelte';

	interface Props {
		/** Most asked first; the server decides the order. */
		gaps: readonly KnowledgeGap[];
		/** Reference time for "first asked 3 d ago"; pass the injected clock's now. */
		now: Date;
		/** Form action URL, e.g. "?/gap". Posts `id` and `decision` (add-fact-check or dismiss). */
		action: string;
		enhance?: FormEnhance;
		emptyText?: string;
		class?: string;
	}

	let {
		gaps,
		now,
		action,
		enhance,
		emptyText = 'Every claim matched the knowledge base.',
		class: className
	}: Props = $props();
</script>

{#if gaps.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<ol aria-label="Knowledge base gaps" class={className}>
		{#each gaps as gap (gap.id)}
			{@const decision = gap.review.kind === 'decided' ? gap.review.decision : undefined}
			<li
				data-decision={decision}
				class="flex flex-wrap items-center gap-x-4 gap-y-2.5 border-t border-rule py-3 transition-opacity data-[decision=dismiss]:opacity-55"
			>
				<div class="flex min-w-0 flex-[1_1_17.5rem] flex-col gap-[0.1875rem]">
					<h3 class="font-sans text-sm/[1.3] font-semibold text-pretty">{gap.label}</h3>
					<blockquote
						class="text-[0.78125rem]/[1.4] text-pretty text-verdict-unverified-foreground italic"
					>
						“{gap.example}”
					</blockquote>
					<p class="font-mono text-[0.6875rem] font-medium text-faint-foreground">
						{gap.languages.join(', ')} · first asked
						<time datetime={gap.firstAskedAt.toISOString()}>{formatAgo(gap.firstAskedAt, now)}</time
						>
					</p>
				</div>
				<p class="w-16 text-right">
					<span class="block font-mono text-[1.0625rem] font-semibold">
						{formatNumber(gap.messageCount)}
					</span>
					<span class="block text-[0.65625rem] text-faint-foreground">messages</span>
				</p>
				<ReviewActions
					itemId={gap.id}
					itemLabel={gap.label}
					state={gap.review}
					{action}
					{enhance}
					decisions={['add-fact-check', 'dismiss']}
				/>
			</li>
		{/each}
	</ol>
{/if}
