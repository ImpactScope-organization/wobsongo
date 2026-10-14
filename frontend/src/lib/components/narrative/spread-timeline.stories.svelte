<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { NARRATIVE_NOW, spread } from '#lib/fixtures/narrative.js';
	import SectionCard from '../common/section-card.svelte';
	import SpreadTimeline from './spread-timeline.svelte';

	const { Story } = defineMeta({
		title: 'Narrative/SpreadTimeline',
		component: SpreadTimeline,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { spread, now: NARRATIVE_NOW }
	});
</script>

<Story
	name="In a card"
	play={async ({ canvasElement }) => {
		const chart = within(canvasElement).getByRole('img', { name: /Reached WhatsApp .*18 h later/ });
		await expect(chart.querySelectorAll('path.lc-path')).toHaveLength(3);
		// Scales with the card instead of overflowing it (LayerChart's wrappers are fixed-size).
		const svg = chart.querySelector('svg')!.getBoundingClientRect();
		const box = chart.getBoundingClientRect();
		await expect(Math.abs(svg.width - box.width)).toBeLessThan(1);
		await expect(Math.abs(svg.height - box.height)).toBeLessThan(1);
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="How it spread"
			description="Mentions per hour by platform, last 72 h"
			class="max-w-3xl"
		>
			<SpreadTimeline {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Narrow"
	play={async ({ canvasElement }) => {
		const chart = within(canvasElement).getByRole('img', { name: /Mentions per hour/ });
		const svg = chart.querySelector('svg')!.getBoundingClientRect();
		await expect(Math.abs(svg.width - chart.getBoundingClientRect().width)).toBeLessThan(1);
		await expect(chart.getBoundingClientRect().width).toBeLessThan(330);
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="How it spread"
			description="Mentions per hour by platform, last 72 h"
			class="w-92"
		>
			<SpreadTimeline {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Not on WhatsApp yet"
	args={{
		spread: {
			...spread,
			series: spread.series.filter((series) => series.platform !== 'whatsapp'),
			firstOnWhatsApp: null
		}
	}}
	play={async ({ canvasElement }) => {
		await expect(
			within(canvasElement).getByRole('img', { name: /Not on WhatsApp yet/ })
		).toBeVisible();
	}}
/>
