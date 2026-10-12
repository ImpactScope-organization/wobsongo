<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { REVIEW_STATUSES, VERDICTS } from '#lib/domain/narrative.js';
	import { toCount } from '#lib/domain/count.js';
	import CountValue from './count-value.svelte';
	import HarmScore from './harm-score.svelte';
	import ReviewStatusBadge from './review-status-badge.svelte';
	import VerdictBadge from './verdict-badge.svelte';

	const { Story } = defineMeta({
		title: 'Common/Badges',
		component: VerdictBadge,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { verdict: 'false' },
		argTypes: { verdict: { control: 'select', options: VERDICTS } }
	});
</script>

<Story name="Verdict" />

<Story
	name="All verdicts"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('False')).toBeVisible();
		await expect(canvas.getByText('Misleading')).toBeVisible();
		await expect(canvas.getByText('Unverified')).toBeVisible();
	}}
>
	{#snippet template()}
		<div class="flex gap-2">
			{#each VERDICTS as verdict (verdict)}
				<VerdictBadge {verdict} />
			{/each}
		</div>
	{/snippet}
</Story>

<Story name="Review statuses">
	{#snippet template()}
		<div class="flex flex-wrap gap-2">
			{#each REVIEW_STATUSES as status (status)}
				<ReviewStatusBadge {status} class="w-28" />
			{/each}
		</div>
	{/snippet}
</Story>

<Story name="Harm scores">
	{#snippet template()}
		<div class="flex flex-col gap-3">
			<div class="flex gap-4">
				<HarmScore harm={2} />
				<HarmScore harm={3} />
				<HarmScore harm={4} />
				<HarmScore harm={5} />
			</div>
			<div class="flex gap-6">
				<HarmScore harm={3} size="large" />
				<HarmScore harm={5} size="large" />
			</div>
		</div>
	{/snippet}
</Story>

<Story
	name="Counts with privacy suppression"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('3,120')).toBeVisible();
		await expect(canvas.getByTitle(/hidden to protect privacy/)).toHaveTextContent('< 10');
	}}
>
	{#snippet template()}
		<dl class="grid w-60 grid-cols-[1fr_auto] gap-x-4 gap-y-1 text-sm">
			<dt>Centre</dt>
			<dd><CountValue count={toCount(3120)} /></dd>
			<dt>Sahel</dt>
			<dd><CountValue count={toCount(6)} /></dd>
		</dl>
	{/snippet}
</Story>
