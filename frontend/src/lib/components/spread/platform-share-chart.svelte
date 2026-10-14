<script lang="ts">
	import { PLATFORMS, type Platform } from '#lib/domain/platform.js';
	import type { WeeklyCounts } from '#lib/domain/spread.js';
	import { formatIsoWeek } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { PLATFORM_LABEL } from '../common/labels';
	import ShareColumns from '../common/share-columns.svelte';
	import SwatchLegend from '../common/swatch-legend.svelte';
	import { PLATFORM_TONE } from '../common/tones';

	interface Props {
		/** Oldest first. */
		weeks: readonly WeeklyCounts<Platform>[];
		class?: string;
	}

	let { weeks, class: className }: Props = $props();
</script>

<div class={cn('flex flex-col gap-4', className)}>
	<SwatchLegend
		items={PLATFORMS.map((platform) => ({
			key: platform,
			label: PLATFORM_LABEL[platform],
			class: PLATFORM_TONE[platform].swatch
		}))}
	/>
	<ShareColumns
		caption="Share of incoming claims by platform, weekly"
		columns={weeks.map((week) => ({
			key: week.weekStart.toISOString(),
			label: formatIsoWeek(week.weekStart),
			segments: PLATFORMS.map((platform) => ({
				key: platform,
				label: PLATFORM_LABEL[platform],
				value: week.counts[platform],
				class: PLATFORM_TONE[platform].swatch
			}))
		}))}
	/>
</div>
