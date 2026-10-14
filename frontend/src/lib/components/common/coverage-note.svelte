<script lang="ts">
	import type { Snippet } from 'svelte';
	import InfoIcon from '@lucide/svelte/icons/info';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** Defaults to the standard monitoring-coverage caveat. */
		children?: Snippet;
		class?: string;
	}

	let { children, class: className }: Props = $props();
</script>

<p
	class={cn(
		'flex gap-1.5 border-t border-dashed pt-3 text-xs/normal text-pretty text-muted-foreground',
		className
	)}
>
	<InfoIcon class="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
	<span>
		{#if children}
			{@render children()}
		{:else}
			Reflects monitoring coverage, not true prevalence. Social figures come from crawled public
			groups and accounts; WhatsApp counts are bot submissions.
		{/if}
	</span>
</p>
