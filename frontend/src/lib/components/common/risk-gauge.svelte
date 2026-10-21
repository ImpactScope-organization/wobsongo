<script lang="ts">
	import { cn } from '#lib/utils.js';

	interface Props {
		/** Risk score from 0 to 100. */
		value: number;
		/** Outer diameter of the radial gauge in pixels. Defaults to 36. */
		size?: number;
		/** Stroke width in pixels. Defaults to 3. */
		strokeWidth?: number;
		/** Optional extra CSS classes for the container. */
		class?: string;
		/** Optional label for accessibility and tooltip. */
		label?: string;
	}

	let {
		value,
		size = 36,
		strokeWidth = 3,
		class: className,
		label = 'Risk level'
	}: Props = $props();

	const clamped = $derived(Math.max(0, Math.min(100, Math.round(value))));

	// Red (high: >= 70), Orange (medium: 40-69), Green (low: < 40)
	const tier = $derived<'low' | 'medium' | 'high'>(
		clamped >= 70 ? 'high' : clamped >= 40 ? 'medium' : 'low'
	);

	const colors = $derived(
		{
			high: {
				stroke: 'var(--verdict-false-strong, #dc2626)',
				text: 'text-verdict-false-strong dark:text-red-400',
				bg: 'fill-red-500/10 dark:fill-red-950/25',
				title: 'High'
			},
			medium: {
				stroke: 'var(--verdict-misleading-strong, #ea580c)',
				text: 'text-verdict-misleading-foreground dark:text-amber-400',
				bg: 'fill-amber-500/10 dark:fill-amber-950/25',
				title: 'Medium'
			},
			low: {
				stroke: 'var(--verdict-accurate-strong, #16a34a)',
				text: 'text-verdict-accurate-strong dark:text-emerald-400',
				bg: 'fill-emerald-500/10 dark:fill-emerald-950/25',
				title: 'Low'
			}
		}[tier]
	);

	const radius = $derived((size - strokeWidth) / 2);
	const circumference = $derived(2 * Math.PI * radius);
	const dashOffset = $derived(circumference * (1 - clamped / 100));
	const fontSize = $derived(Math.max(9, Math.round(size * 0.31)));
</script>

<div
	class={cn('relative inline-flex shrink-0 items-center justify-center select-none', className)}
	style="width: {size}px; height: {size}px;"
	title="{label}: {clamped}/100 ({colors.title})"
>
	<svg
		width={size}
		height={size}
		viewBox="0 0 {size} {size}"
		class="overflow-visible"
		role="meter"
		aria-valuenow={clamped}
		aria-valuemin="0"
		aria-valuemax="100"
		aria-label="{label}: {clamped}/100 ({colors.title})"
	>
		<!-- Background circle with subtle soft tint -->
		<circle cx={size / 2} cy={size / 2} r={radius} class={colors.bg} />
		<!-- Track circle -->
		<circle
			cx={size / 2}
			cy={size / 2}
			r={radius}
			fill="none"
			stroke="currentColor"
			class="text-border/60 dark:text-muted/60"
			stroke-width={strokeWidth}
		/>
		<!-- Progress circle (radial) -->
		{#if clamped > 0}
			<circle
				cx={size / 2}
				cy={size / 2}
				r={radius}
				fill="none"
				stroke={colors.stroke}
				stroke-width={strokeWidth}
				stroke-linecap="round"
				stroke-dasharray={circumference}
				stroke-dashoffset={dashOffset}
				transform="rotate(-90 {size / 2} {size / 2})"
				class="transition-[stroke-dashoffset] duration-300"
			/>
		{/if}
		<!-- Centered number: 0 to 100 -->
		<text
			x={size / 2}
			y={size / 2}
			text-anchor="middle"
			dominant-baseline="central"
			class={cn('font-mono leading-none font-bold', colors.text)}
			style="font-size: {fontSize}px;"
			fill="currentColor"
		>
			{clamped}
		</text>
	</svg>
</div>
