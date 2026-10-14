<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { activity, PULSE_NOW } from '#lib/fixtures/pulse.js';
	import ActivityLog from './activity-log.svelte';
	import CoverageNote from './coverage-note.svelte';
	import SectionCard from './section-card.svelte';

	const { Story } = defineMeta({
		title: 'Common/SectionCard',
		component: SectionCard,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: {
			title: 'Where claims come from',
			description: 'Incoming claims by source platform'
		}
	});
</script>

<Story
	name="With coverage note"
	play={async ({ canvasElement }) => {
		const region = within(canvasElement).getByRole('region', { name: 'Where claims come from' });
		await expect(region).toHaveTextContent('not true prevalence');
	}}
>
	{#snippet template(args)}
		<SectionCard title={args.title} description={args.description} class="max-w-md">
			<p class="text-sm">Section body.</p>
			{#snippet footer()}
				<CoverageNote />
			{/snippet}
		</SectionCard>
	{/snippet}
</Story>

<Story name="Activity log" args={{ title: 'Activity log', description: undefined }}>
	{#snippet template(args)}
		<SectionCard title={args.title} description={args.description} class="max-w-xl">
			<ActivityLog entries={activity} now={PULSE_NOW} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Empty activity log" args={{ title: 'Activity log', description: undefined }}>
	{#snippet template(args)}
		<SectionCard title={args.title} description={args.description} class="max-w-xl">
			<ActivityLog entries={[]} now={PULSE_NOW} />
		</SectionCard>
	{/snippet}
</Story>
