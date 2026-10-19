<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { messagesTrend, ratedBreakdown } from '#lib/fixtures/overview.js';
	import KpiGroup from './kpi-group.svelte';
	import KpiTile from './kpi-tile.svelte';
	import Sparkline from './sparkline.svelte';
	import VerdictMeter from './verdict-meter.svelte';

	const { Story } = defineMeta({
		title: 'Common/KPI tiles',
		component: KpiTile,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { label: 'Active narratives', value: '37', caption: '6 new today' }
	});
</script>

<Story name="Single tile">
	{#snippet template(args)}
		<KpiGroup class="max-w-xs">
			<KpiTile {...args} />
		</KpiGroup>
	{/snippet}
</Story>

<Story
	name="Overview strip"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getAllByRole('term')).toHaveLength(4);
		await expect(canvas.getByRole('img', { name: /14 false, 7 misleading/ })).toBeVisible();
	}}
>
	{#snippet template()}
		<KpiGroup>
			<KpiTile
				label="Messages received, 24 h"
				value="8,412"
				caption="▲ 64% vs prev. 24 h"
				captionTone="signal"
			>
				{#snippet aside()}
					<Sparkline
						values={messagesTrend}
						label="Messages per hour, rising"
						width={120}
						height={36}
						endDot
					/>
				{/snippet}
			</KpiTile>
			<KpiTile label="Active narratives" value="37" caption="6 new today" />
			<KpiTile label="Rated false or misleading" value="21" suffix="/ 37">
				<VerdictMeter counts={ratedBreakdown} />
			</KpiTile>
			<KpiTile
				label="Median time to debunk"
				value="142"
				suffix="min"
				caption="▲ 18 min vs last week"
				captionTone="signal"
			/>
		</KpiGroup>
	{/snippet}
</Story>

<Story name="Verdict meter, nothing rated">
	{#snippet template()}
		<VerdictMeter class="w-48" counts={{ false: 0, misleading: 0, unverified: 0, accurate: 0 }} />
	{/snippet}
</Story>
