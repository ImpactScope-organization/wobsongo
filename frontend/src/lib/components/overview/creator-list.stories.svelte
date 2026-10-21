<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { sampleCreators } from '#lib/fixtures/overview.js';
	import SectionCard from '../common/section-card.svelte';
	import CreatorList from './creator-list.svelte';

	const { Story } = defineMeta({
		title: 'Overview/CreatorList',
		component: CreatorList,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: {
			creators: sampleCreators,
			hasAccess: true
		}
	});
</script>

<Story
	name="Full data"
	args={{ privacyMode: false }}
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const list = await canvas.findByRole('list', { name: 'Content creators' });
		await expect(list).toBeInTheDocument();

		const items = within(list).getAllByRole('listitem');
		await expect(items).toHaveLength(5);
		await expect(within(list).getAllByText(/Aïcha Ouédraogo/).length).toBeGreaterThan(0);
		await expect(within(list).getAllByText(/@aicha_oued/).length).toBeGreaterThan(0);
		await expect(within(list).getAllByText(/890K/).length).toBeGreaterThan(0);
		await expect(within(list).getAllByText(/52/).length).toBeGreaterThan(0);
		await expect(
			within(list).getAllByRole('link', { name: /Aïcha Ouédraogo/ }).length
		).toBeGreaterThan(0);
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="Content creators"
			description="Unredacted view with full creator metrics and links"
			flush
			class="max-w-4xl"
		>
			<CreatorList {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Redacted data"
	args={{ privacyMode: true }}
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const list = await canvas.findByRole('list', { name: 'Content creators' });
		await expect(list).toBeInTheDocument();

		const items = within(list).getAllByRole('listitem');
		await expect(items).toHaveLength(5);
		await expect(within(list).getAllByText(/Creator 01/).length).toBeGreaterThan(0);
		await expect(within(list).queryAllByText(/Aïcha Ouédraogo/)).toHaveLength(0);
		await expect(within(list).queryAllByText(/@aicha_oued/)).toHaveLength(0);
		await expect(within(list).getAllByText(/77%/).length).toBeGreaterThan(0);
		await expect(within(list).queryAllByRole('link')).toHaveLength(0);
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="Content creators"
			description="Privacy-protected view with pseudonyms and masked statistics"
			flush
			class="max-w-4xl"
		>
			<CreatorList {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Empty list"
	args={{ creators: [] }}
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(
			await canvas.findByText('No content creators detected in this window.')
		).toBeInTheDocument();
	}}
>
	{#snippet template(args)}
		<SectionCard title="Content creators" flush class="max-w-4xl">
			<CreatorList {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Mobile"
	args={{ privacyMode: true }}
	globals={{ viewport: { value: 'mobile1' } }}
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const list = await canvas.findByRole('list', { name: 'Content creators' });
		await expect(list).toBeInTheDocument();

		const items = within(list).getAllByRole('listitem');
		await expect(items).toHaveLength(5);
		await expect(within(list).getAllByText(/Creator 01/).length).toBeGreaterThan(0);
	}}
>
	{#snippet template(args)}
		<SectionCard title="Content creators" flush>
			<CreatorList {...args} />
		</SectionCard>
	{/snippet}
</Story>
