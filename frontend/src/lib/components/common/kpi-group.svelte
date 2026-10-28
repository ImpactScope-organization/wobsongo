<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '#lib/utils.js';

	interface Props {
		/**
		 * Accessible name for the strip, e.g. "Key figures, last 24 h", read when a
		 * screen reader enters it. Give one whenever the page has more than one
		 * strip or the tiles need context.
		 */
		label?: string;
		/** `KpiTile`s. */
		children: Snippet;
		class?: string;
	}

	let { label, children, class: className }: Props = $props();
</script>

<!--
	Tiles sit on a 1px border-coloured gap, so they read as one ruled strip.
	Each tile is a term (its label) with its definitions (value, visual,
	caption). With a label the list is a named group, which a bare <dl> can't be.
-->
<dl
	role={label ? 'group' : undefined}
	aria-label={label}
	class={cn(
		'flex flex-wrap gap-px overflow-hidden rounded-lg border bg-border *:flex-[1_1_12.5rem]',
		className
	)}
>
	{@render children()}
</dl>
