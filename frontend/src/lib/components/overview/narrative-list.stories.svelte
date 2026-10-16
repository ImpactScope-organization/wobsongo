<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { narratives } from '#lib/fixtures/overview.js';
	import SectionCard from '../common/section-card.svelte';
	import SegmentedLinks from '../common/segmented-links.svelte';
	import NarrativeList from './narrative-list.svelte';

	const { Story } = defineMeta({
		title: 'Overview/NarrativeList',
		component: NarrativeList,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { narratives }
	});
</script>

<Story
	name="Top narratives"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const list = canvas.getByRole('list', { name: 'Narratives' });
		const items = within(list).getAllByRole('listitem');
		await expect(items).toHaveLength(5);
		await expect(
			canvas.getByRole('link', { name: 'Compulsory contraceptive trial for schoolgirls' })
		).toHaveAttribute('href', '/narratives/N-0412');
		await expect(canvas.getAllByRole('img', { name: /spiking/ })).toHaveLength(3);
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="Top misinformation narratives"
			description="Clustered claims, last 24 h, ranked by volume"
			flush
			class="max-w-4xl"
		>
			{#snippet actions()}
				<SegmentedLinks
					label="Topic"
					variant="pills"
					current="misinfo"
					options={[
						{ value: 'misinfo', label: 'Misinformation', href: '?topic=misinfo' },
						{ value: 'all', label: 'All topics', href: '?topic=all' }
					]}
				/>
			{/snippet}
			<NarrativeList {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Narrow (phone)" globals={{ viewport: { value: 'mobile1' } }}>
	{#snippet template(args)}
		<SectionCard title="Top misinformation narratives" flush>
			<NarrativeList {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Empty"
	args={{ narratives: [] }}
	play={async ({ canvasElement }) => {
		await expect(within(canvasElement).getByText('No narratives in this window.')).toBeVisible();
	}}
/>
