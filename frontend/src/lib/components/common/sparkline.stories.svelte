<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import Sparkline from './sparkline.svelte';

	const { Story } = defineMeta({
		title: 'Common/Sparkline',
		component: Sparkline,
		tags: ['autodocs'],
		args: {
			values: [1, 1, 1, 2, 1, 2, 2, 3, 4, 7, 12, 20],
			label: 'Volume over the last 24 hours, spiking',
			tone: 'signal'
		},
		argTypes: { tone: { control: 'inline-radio', options: ['default', 'muted', 'signal'] } }
	});
</script>

<Story
	name="Spiking"
	play={async ({ canvasElement }) => {
		const chart = within(canvasElement).getByRole('img', { name: /spiking/ });
		await expect(chart.querySelector('path')).not.toBeNull();
	}}
/>

<Story
	name="Steady"
	args={{
		values: [9, 10, 9, 9, 10, 9, 10, 10, 9, 10, 11, 10],
		tone: 'muted',
		label: 'Volume, steady'
	}}
/>

<Story
	name="KPI size with end dot"
	args={{
		values: [30, 31, 29, 32, 31, 33, 32, 34, 33, 38, 46, 52, 57],
		tone: 'default',
		endDot: true,
		width: 120,
		height: 36
	}}
/>

<Story
	name="Not enough data"
	args={{ values: [4], label: 'Volume, not enough data' }}
	play={async ({ canvasElement }) => {
		const chart = within(canvasElement).getByRole('img', { name: /not enough data/ });
		await expect(chart.querySelector('path')).toBeNull();
	}}
/>
