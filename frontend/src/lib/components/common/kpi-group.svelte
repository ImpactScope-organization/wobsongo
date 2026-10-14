<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** `KpiTile`s. */
		children: Snippet;
		class?: string;
	}

	let { children, class: className }: Props = $props();
</script>

<!-- Tiles sit on a 1px border-coloured gap, so they read as one ruled strip. -->
<dl
	class={cn(
		'flex flex-wrap gap-px overflow-hidden rounded-lg border bg-border *:flex-[1_1_12.5rem]',
		className
	)}
>
	{@render children()}
</dl>
