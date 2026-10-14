<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { highHarmSummary } from '#lib/fixtures/pulse.js';
	import HighHarmSummary from './high-harm-summary.svelte';

	const { Story } = defineMeta({
		title: 'Pulse/HighHarmSummary',
		component: HighHarmSummary,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { summary: highHarmSummary, href: '/safety', class: 'max-w-sm' }
	});
</script>

<Story
	name="Default"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('link', { name: 'High-harm claims' })).toHaveAttribute(
			'href',
			'/safety'
		);
		await expect(canvas.getByText('claim fast-tracked to the fact-check queue')).toBeVisible();
	}}
/>

<Story name="Nothing pending" args={{ summary: { awaitingReview: 0, fastTracked: 0 } }} />
