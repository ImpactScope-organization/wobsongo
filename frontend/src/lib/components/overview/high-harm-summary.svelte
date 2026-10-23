<script lang="ts">
	import type { HighHarmSummary } from '#lib/domain/narrative.js';
	import { m } from '#lib/paraglide/messages.js';
	import { plural } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		summary: HighHarmSummary;
		/** Link to the high-harm view. */
		href: string;
		class?: string;
	}

	let { summary, href, class: className }: Props = $props();

	const rows = $derived([
		{
			count: summary.awaitingReview,
			text: plural(
				summary.awaitingReview,
				m.overview_high_harm_awaiting_one(),
				m.overview_high_harm_awaiting_other()
			)
		},
		{
			count: summary.fastTracked,
			text: plural(
				summary.fastTracked,
				m.overview_high_harm_fast_tracked_one(),
				m.overview_high_harm_fast_tracked_other()
			)
		}
	]);

	const headingId = $props.id();
</script>

<section
	aria-labelledby={headingId}
	class={cn(
		'relative rounded-lg bg-inverse px-5 py-[1.125rem] text-inverse-foreground hover:bg-inverse-raised has-focus-visible:ring-2 has-focus-visible:ring-inverse-signal',
		className
	)}
>
	<div class="mb-3 flex items-center justify-between gap-2.5">
		<h2 id={headingId} class="text-lg font-bold">
			<!-- The heading link's ::after makes the whole card clickable. -->
			<a {href} class="outline-none after:absolute after:inset-0">{m.overview_high_harm_title()}</a>
		</h2>
		<span
			class="rounded border border-muted-foreground px-1.5 py-0.5 font-mono text-[0.625rem] font-semibold tracking-[0.06em] text-inverse-muted-foreground uppercase"
		>
			{m.overview_high_harm_priority()}
		</span>
	</div>

	<ul class="flex flex-col gap-px overflow-hidden rounded-md bg-inverse-border">
		{#each rows as row (row.text)}
			<li class="flex items-center gap-3 bg-inverse-raised px-3.5 py-3">
				<span class="w-7 font-mono text-[1.375rem] font-semibold text-inverse-signal"
					>{row.count}</span
				>
				<span class="text-[0.8125rem]/snug">{row.text}</span>
			</li>
		{/each}
	</ul>

	<p class="mt-3 text-xs/normal text-inverse-muted-foreground">
		{m.overview_high_harm_note()}
	</p>
</section>
