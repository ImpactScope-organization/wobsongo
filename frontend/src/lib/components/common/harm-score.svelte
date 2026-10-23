<script lang="ts">
	import { isHighHarm, MAX_HARM, type HarmScore } from '#lib/domain/narrative.js';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		harm: HarmScore;
		/** `inline` for metadata rows, `large` for metric tiles. */
		size?: 'inline' | 'large';
		class?: string;
	}

	let { harm, size = 'inline', class: className }: Props = $props();
</script>

{#if size === 'large'}
	<span
		data-high-harm={isHighHarm(harm) || undefined}
		class={cn(
			'font-mono text-[2rem] leading-none font-semibold tracking-tight data-high-harm:text-signal',
			className
		)}
	>
		{harm}<span class="text-base text-muted-foreground">/{MAX_HARM}</span>
	</span>
{:else}
	<span
		data-high-harm={isHighHarm(harm) || undefined}
		class={cn(
			'text-[0.71875rem] font-medium text-muted-foreground data-high-harm:text-signal',
			className
		)}
	>
		{m.common_harm_score({ score: harm, max: MAX_HARM })}
	</span>
{/if}
