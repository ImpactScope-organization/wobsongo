<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';
	import { MAX_HARM, isSpiking } from '#lib/domain/narrative.js';
	import type { NarrativeDetail } from '#lib/domain/narrative-detail.js';
	import { formatFactor, formatNumber } from '#lib/utils/format.js';
	import KpiGroup from '../common/kpi-group.svelte';
	import KpiTile from '../common/kpi-tile.svelte';

	interface Props {
		narrative: Pick<
			NarrativeDetail,
			'volume7d' | 'growthFactor6h' | 'harm' | 'harmValidated' | 'priority'
		>;
		class?: string;
	}

	let { narrative, class: className }: Props = $props();

	const spiking = $derived(isSpiking((narrative.growthFactor6h - 1) * 100));
</script>

<KpiGroup class={className}>
	<KpiTile size="sm" label="Messages, 7 d" value={formatNumber(narrative.volume7d)} />
	<KpiTile
		size="sm"
		label="Growth, last 6 h"
		value={formatFactor(narrative.growthFactor6h)}
		valueTone={spiking ? 'signal' : 'default'}
	/>
	<KpiTile size="sm" label="Harm score" value={String(narrative.harm)} suffix="/{MAX_HARM}">
		{#snippet aside()}
			{#if narrative.harmValidated}
				<span
					class="mr-auto ml-2 flex items-center gap-1 self-end text-[0.6875rem] font-medium text-verdict-accurate-strong"
				>
					<CheckIcon class="size-3" aria-hidden="true" />
					Expert-validated
				</span>
			{/if}
		{/snippet}
	</KpiTile>
	<KpiTile size="sm" label="Priority score" value={String(narrative.priority)} />
</KpiGroup>
