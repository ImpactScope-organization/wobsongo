<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** The headline finding, in bold, e.g. "Voice notes now 52%." */
		title: string;
		/** What it means or what to do about it. */
		children?: Snippet;
		class?: string;
	}

	let { title, children, class: className }: Props = $props();
</script>

<p
	class={cn(
		'rounded-lg bg-inverse px-3.5 py-3 text-[0.8125rem]/[1.45] text-pretty text-inverse-foreground',
		className
	)}
>
	<strong class="font-semibold">{title}</strong>
	{#if children}{@render children()}{/if}
</p>
