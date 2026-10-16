<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { platformCounts } from '#lib/fixtures/overview.js';
	import CoverageNote from '../common/coverage-note.svelte';
	import SectionCard from '../common/section-card.svelte';
	import PlatformBreakdown from './platform-breakdown.svelte';

	const { Story } = defineMeta({
		title: 'Overview/PlatformBreakdown',
		component: PlatformBreakdown,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { counts: platformCounts }
	});
</script>

<Story
	name="In a card"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(
			canvas.getByRole('img', { name: /Claims by platform: WhatsApp \(bot\) 46%/ })
		).toBeVisible();
		const row = canvas.getByRole('row', { name: /Facebook/ });
		await expect(row).toHaveTextContent('1,571');
		await expect(row).toHaveTextContent('27%');
		await expect(canvasElement.querySelectorAll('.lc-arc-line')).toHaveLength(5);
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="Where claims come from"
			description="Incoming claims by source platform"
			class="max-w-md"
		>
			<PlatformBreakdown {...args} />
			{#snippet footer()}
				<CoverageNote />
			{/snippet}
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="No claims yet"
	args={{ counts: platformCounts.map((item) => ({ ...item, count: 0 })) }}
	play={async ({ canvasElement }) => {
		await expect(canvasElement.querySelectorAll('.lc-arc-line')).toHaveLength(0);
	}}
/>

<Story
	name="One dominant platform"
	args={{
		counts: [
			{ platform: 'whatsapp', count: 980 },
			{ platform: 'facebook', count: 12 },
			{ platform: 'other', count: 8 }
		]
	}}
/>
