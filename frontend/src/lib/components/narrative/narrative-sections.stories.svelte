<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import {
		debunkFormats,
		formats,
		languages,
		narrative,
		regionRespondents,
		regions,
		samples
	} from '#lib/fixtures/narrative.js';
	import { formatTotal } from '#lib/utils/format.js';
	import SectionCard from '../common/section-card.svelte';
	import AudienceBreakdown from './audience-breakdown.svelte';
	import DebunkFormats from './debunk-formats.svelte';
	import NarrativeMetrics from './narrative-metrics.svelte';
	import RegionList from '../common/region-list.svelte';
	import SampleMessages from './sample-messages.svelte';

	const { Story } = defineMeta({
		title: 'Narrative/Sections',
		component: NarrativeMetrics,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { narrative }
	});
</script>

<Story
	name="Metrics"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getAllByRole('term')).toHaveLength(4);
		await expect(canvas.getByText('Expert-validated')).toBeVisible();
	}}
/>

<Story name="Language and format">
	{#snippet template()}
		<SectionCard title="Language" class="max-w-sm">
			<AudienceBreakdown {languages} {formats} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Regions"
	play={async ({ canvasElement }) => {
		const table = within(canvasElement).getByRole('table', { name: 'Messages by region' });
		await expect(within(table).getByRole('row', { name: /Sahel/ })).toHaveTextContent('< 10');
	}}
>
	{#snippet template()}
		<SectionCard
			title="Regions"
			description="Voluntary location answers in the bot, {formatTotal(regionRespondents)}"
			class="max-w-sm"
		>
			<RegionList {regions} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Sample messages">
	{#snippet template()}
		<SectionCard
			title="What people are saying"
			description="Anonymized excerpts. Names, numbers and group names are removed."
			class="max-w-4xl"
		>
			<SampleMessages {samples} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Debunk formats">
	{#snippet template()}
		<SectionCard
			title="Debunk formats"
			description="Suggested from language and format mix"
			class="max-w-sm"
		>
			<DebunkFormats formats={debunkFormats} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Empty states">
	{#snippet template()}
		<div class="grid max-w-3xl grid-cols-2 gap-5">
			<SectionCard title="Regions"><RegionList regions={[]} /></SectionCard>
			<SectionCard title="Debunk formats"><DebunkFormats formats={[]} /></SectionCard>
			<SectionCard title="What people are saying" class="col-span-2">
				<SampleMessages samples={[]} />
			</SectionCard>
		</div>
	{/snippet}
</Story>
