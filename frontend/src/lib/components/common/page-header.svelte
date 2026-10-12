<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '#lib/utils.js';

	interface Props {
		title: string;
		/** Small line above the title, e.g. "Burkina Faso · updated 14:05 GMT". */
		eyebrow?: string;
		/** Right-aligned controls, e.g. a time-range toggle. */
		actions?: Snippet;
		class?: string;
	}

	let { title, eyebrow, actions, class: className }: Props = $props();
</script>

<header class={cn('flex flex-wrap items-end justify-between gap-3', className)}>
	<div>
		{#if eyebrow}
			<p class="text-eyebrow mb-1.5 text-muted-foreground">{eyebrow}</p>
		{/if}
		<h1
			class="text-[clamp(1.625rem,3.4vw,2.375rem)] leading-[1.05] font-extrabold tracking-[-0.025em]"
		>
			{title}
		</h1>
	</div>
	{#if actions}
		<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>
	{/if}
</header>
