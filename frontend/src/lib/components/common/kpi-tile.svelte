<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '#lib/utils.js';

	interface Props {
		label: string;
		/** Already formatted, e.g. "8,412". */
		value: string;
		/** Smaller text after the value, e.g. "/ 37". */
		suffix?: string;
		/** Line under the value, e.g. "▲ 64% vs prev. 24 h". */
		caption?: string;
		/** `signal` colours the caption for changes that need attention. */
		captionTone?: 'muted' | 'signal';
		/** Visual beside the value, e.g. a sparkline. */
		aside?: Snippet;
		/** Visual under the value, in place of a caption, e.g. a meter. */
		children?: Snippet;
		class?: string;
	}

	let {
		label,
		value,
		suffix,
		caption,
		captionTone = 'muted',
		aside,
		children,
		class: className
	}: Props = $props();
</script>

<!-- Must sit inside a <dl> (KpiGroup). -->
<div class={cn('flex flex-col gap-2.5 bg-card px-5 py-[1.125rem]', className)}>
	<dt class="text-xs font-medium text-muted-foreground">{label}</dt>
	<dd class="flex items-end justify-between gap-2.5">
		<span class="font-mono text-[2rem] leading-none font-semibold tracking-[-0.03em]">
			{value}{#if suffix}<span class="text-base text-muted-foreground">&nbsp;{suffix}</span>{/if}
		</span>
		{#if aside}{@render aside()}{/if}
	</dd>
	{#if children}
		<dd>{@render children()}</dd>
	{/if}
	{#if caption}
		<dd
			data-tone={captionTone}
			class="font-mono text-xs font-medium text-muted-foreground data-[tone=signal]:text-signal"
		>
			{caption}
		</dd>
	{/if}
</div>
