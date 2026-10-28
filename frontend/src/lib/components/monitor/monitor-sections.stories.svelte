<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import {
		MONITOR_NOW,
		monitorWindows,
		processingRuns,
		runningRun
	} from '#lib/fixtures/monitor.js';
	import SectionCard from '../common/section-card.svelte';
	import ProcessingVolumeChart from './processing-volume-chart.svelte';
	import RunPlatformBreakdown from './run-platform-breakdown.svelte';
	import RunStatusBreakdown from './run-status-breakdown.svelte';
	import RunTable from './run-table.svelte';

	const { Story } = defineMeta({
		title: 'Monitor/Sections',
		component: RunTable,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { runs: processingRuns, now: MONITOR_NOW, selectedId: processingRuns[0].id }
	});
</script>

<Story
	name="Processing runs"
	play={async ({ canvasElement }) => {
		const table = within(canvasElement).getByRole('table', { name: 'Processing runs' });
		const selected = within(table).getByRole('link', { current: true });
		await expect(selected).toHaveTextContent('Today 14:00');
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="Processing runs"
			description="Automatic every 2 h, or triggered by an administrator."
			flush
		>
			<RunTable {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Processing runs, one running" args={{ runs: [runningRun, ...processingRuns] }} />

<Story name="Processing status">
	{#snippet template()}
		<SectionCard title="Processing status" eyebrow="Today 14:00 · Automatic" class="max-w-sm">
			<RunStatusBreakdown run={processingRuns[0]} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Processing status, running">
	{#snippet template()}
		<SectionCard
			title="Processing status"
			eyebrow="Today 14:07 · Manual · Admin 1"
			class="max-w-sm"
		>
			<RunStatusBreakdown run={runningRun} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Posts by platform"
	play={async ({ canvasElement }) => {
		await expect(canvasElement.querySelectorAll('.lc-arc-line')).toHaveLength(5);
	}}
>
	{#snippet template()}
		<SectionCard
			title="Posts in run Today 14:00"
			description="Newly discovered posts in this run by platform"
			class="max-w-md"
		>
			<RunPlatformBreakdown counts={processingRuns[0].result!.platformCounts} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Posts by platform, running">
	{#snippet template()}
		<SectionCard title="Posts in run Today 14:07" class="max-w-md">
			<RunPlatformBreakdown counts={null} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Processing volume, 24 h">
	{#snippet template()}
		<SectionCard
			title="Processing volume"
			description="Posts successfully processed per run window (2 h)"
		>
			<ProcessingVolumeChart {...monitorWindows['24h'].volume} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Processing volume, 30 d">
	{#snippet template()}
		<SectionCard title="Processing volume" description="Posts successfully processed per day">
			<ProcessingVolumeChart {...monitorWindows['30d'].volume} />
		</SectionCard>
	{/snippet}
</Story>
