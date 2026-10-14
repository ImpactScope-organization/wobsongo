<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import BarList from './bar-list.svelte';
	import StackedBar from './stacked-bar.svelte';

	const formatSegments = [
		{ key: 'audio', label: 'Voice note', value: 52, class: 'bg-primary text-primary-foreground' },
		{ key: 'video', label: 'Video', value: 23, class: 'bg-muted-foreground text-white' },
		{ key: 'text', label: 'Text', value: 17, class: 'bg-verdict-unverified-strong' },
		{ key: 'image', label: 'Image', value: 8, class: 'bg-platform-other' }
	];

	const { Story } = defineMeta({
		title: 'Common/Bars',
		component: StackedBar,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { segments: formatSegments, size: 'lg', describe: 'percent', showPercent: true }
	});
</script>

<Story
	name="Stacked, labelled"
	play={async ({ canvasElement }) => {
		await expect(
			within(canvasElement).getByRole('img', {
				name: 'Voice note 52%, Video 23%, Text 17%, Image 8%'
			})
		).toBeVisible();
	}}
>
	{#snippet template(args)}
		<StackedBar {...args} class="max-w-sm" />
	{/snippet}
</Story>

<Story name="Stacked, thin meter" args={{ size: 'sm', describe: 'count', showPercent: false }}>
	{#snippet template(args)}
		<StackedBar {...args} class="max-w-48" />
	{/snippet}
</Story>

<Story
	name="Bar list"
	play={async ({ canvasElement }) => {
		const table = within(canvasElement).getByRole('table', { name: 'Messages by language' });
		await expect(within(table).getByRole('row', { name: /Mooré/ })).toHaveTextContent('58%');
	}}
>
	{#snippet template()}
		<BarList
			class="max-w-sm"
			caption="Messages by language"
			labelHeader="Language"
			items={[
				{ key: 'moore', label: 'Mooré', value: 1178 },
				{ key: 'french', label: 'French', value: 488 },
				{ key: 'dioula', label: 'Dioula', value: 244 },
				{ key: 'fulfulde', label: 'Fulfulde', value: 81 },
				{ key: 'other', label: 'Other', value: 40 }
			]}
		/>
	{/snippet}
</Story>
