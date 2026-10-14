<script lang="ts">
	import { MESSAGE_FORMATS, type MessageFormat } from '#lib/domain/narrative-detail.js';
	import type { WeeklyCounts } from '#lib/domain/spread.js';
	import { formatIsoWeek } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { MESSAGE_FORMAT_LABEL } from '../common/labels';
	import ShareColumns from '../common/share-columns.svelte';
	import SwatchLegend from '../common/swatch-legend.svelte';
	import { FORMAT_TONE } from '../common/tones';

	interface Props {
		/** Oldest first. */
		weeks: readonly WeeklyCounts<MessageFormat>[];
		class?: string;
	}

	let { weeks, class: className }: Props = $props();
</script>

<div class={cn('flex flex-col gap-2.5', className)}>
	<ShareColumns
		caption="Share of claims by format, weekly"
		barsClass="h-40"
		columns={weeks.map((week) => ({
			key: week.weekStart.toISOString(),
			label: formatIsoWeek(week.weekStart),
			segments: MESSAGE_FORMATS.map((format) => ({
				key: format,
				label: MESSAGE_FORMAT_LABEL[format],
				value: week.counts[format],
				class: FORMAT_TONE[format].swatch
			}))
		}))}
	/>
	<SwatchLegend
		items={MESSAGE_FORMATS.map((format) => ({
			key: format,
			label: MESSAGE_FORMAT_LABEL[format],
			class: FORMAT_TONE[format].swatch
		}))}
	/>
</div>
