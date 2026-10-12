<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '#lib/utils.js';

	interface Props {
		title: string;
		description?: string;
		/** Heading level; keep the page outline correct. */
		level?: 2 | 3;
		/** Right-aligned controls in the header, e.g. filters. */
		actions?: Snippet;
		/** Body. Pass `flush` when it draws its own row padding (lists, tables). */
		children: Snippet;
		flush?: boolean;
		/** Notes under the body, e.g. a coverage caveat. */
		footer?: Snippet;
		class?: string;
	}

	let {
		title,
		description,
		level = 2,
		actions,
		children,
		flush = false,
		footer,
		class: className
	}: Props = $props();

	const headingId = $props.id();
</script>

<section
	aria-labelledby={headingId}
	class={cn('min-w-0 overflow-hidden rounded-lg border bg-card text-card-foreground', className)}
>
	<header class="flex flex-wrap items-baseline justify-between gap-3 px-5 pt-[1.125rem] pb-3.5">
		<div class="min-w-0">
			<svelte:element
				this={`h${level}`}
				id={headingId}
				class="text-lg font-bold tracking-[-0.01em]"
			>
				{title}
			</svelte:element>
			{#if description}
				<p class="mt-1 text-[0.8125rem] text-muted-foreground">{description}</p>
			{/if}
		</div>
		{#if actions}
			<div class="flex flex-wrap gap-1.5">{@render actions()}</div>
		{/if}
	</header>

	<div class={cn(!flush && 'px-5 pb-[1.125rem]')}>
		{@render children()}
	</div>

	{#if footer}
		<div class="px-5 pb-4">{@render footer()}</div>
	{/if}
</section>
