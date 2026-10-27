<script lang="ts">
	import type { Snippet } from 'svelte';
	import { m } from '#lib/paraglide/messages.js';
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
		/** `sm` for secondary metric strips, e.g. on detail pages. */
		size?: 'default' | 'sm';
		/** `signal` colours the value itself, e.g. a spiking growth figure. */
		valueTone?: 'default' | 'signal';
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
		size = 'default',
		valueTone = 'default',
		class: className
	}: Props = $props();

	// A leading ▲ or ▼ is read aloud as "black up-pointing triangle", so screen
	// readers get a word instead.
	const trend = $derived.by(() => {
		const match = caption?.match(/^([▲▼])\s*(.*)$/s);
		if (!match) return undefined;
		const [, arrow, rest] = match;
		return { arrow, rest, word: arrow === '▲' ? m.common_trend_up() : m.common_trend_down() };
	});
</script>

<!-- Must sit inside a <dl> (KpiGroup). -->
<div
	data-size={size}
	class={cn(
		'group/kpi flex flex-col gap-2.5 bg-card px-5 py-[1.125rem] data-[size=sm]:gap-2 data-[size=sm]:px-[1.125rem] data-[size=sm]:py-4',
		className
	)}
>
	<dt class="text-xs font-medium text-muted-foreground">{label}</dt>
	<dd class="flex items-end justify-between gap-2.5">
		<span
			data-tone={valueTone}
			class="font-mono text-[2rem] leading-none font-semibold tracking-[-0.03em] group-data-[size=sm]/kpi:text-[1.625rem] group-data-[size=sm]/kpi:tracking-normal data-[tone=signal]:text-signal"
		>
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
			{#if trend}
				<span aria-hidden="true">{trend.arrow}</span><span class="sr-only">{trend.word}</span>
				{trend.rest}
			{:else}
				{caption}
			{/if}
		</dd>
	{/if}
</div>
