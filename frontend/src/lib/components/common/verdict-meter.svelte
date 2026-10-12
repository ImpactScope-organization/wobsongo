<script lang="ts" module>
	/** Number of rated narratives per outcome. */
	export interface VerdictCounts {
		false: number;
		misleading: number;
		unverified: number;
		accurate: number;
	}
</script>

<script lang="ts">
	import { cn } from '#lib/utils.js';

	interface Props {
		counts: VerdictCounts;
		class?: string;
	}

	let { counts, class: className }: Props = $props();

	// A proportion meter rather than a chart: one row of flex segments, no JS.
	const segments = $derived(
		[
			{ key: 'false', label: 'false', count: counts.false, class: 'bg-verdict-false-strong' },
			{
				key: 'misleading',
				label: 'misleading',
				count: counts.misleading,
				class: 'bg-verdict-misleading-strong'
			},
			{
				key: 'unverified',
				label: 'unverified',
				count: counts.unverified,
				class: 'bg-verdict-unverified-strong'
			},
			{
				key: 'accurate',
				label: 'accurate',
				count: counts.accurate,
				class: 'bg-verdict-accurate-strong'
			}
		].filter((segment) => segment.count > 0)
	);

	const description = $derived(
		segments.map((segment) => `${segment.count} ${segment.label}`).join(', ') ||
			'No rated narratives'
	);
</script>

<div
	role="img"
	aria-label={description}
	title={description}
	class={cn('flex h-1.5 gap-0.5 overflow-hidden rounded-full bg-secondary', className)}
>
	{#each segments as segment (segment.key)}
		<div data-segment={segment.key} class={segment.class} style:flex-grow={segment.count}></div>
	{/each}
</div>
