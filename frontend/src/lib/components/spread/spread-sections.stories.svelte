<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import {
		dailyIntents,
		formatWeeks,
		languageFormats,
		lags,
		platformWeeks
	} from '#lib/fixtures/spread.js';
	import CoverageNote from '../common/coverage-note.svelte';
	import InsightCallout from '../common/insight-callout.svelte';
	import SectionCard from '../common/section-card.svelte';
	import FormatShareChart from './format-share-chart.svelte';
	import IntentBreakdown from './intent-breakdown.svelte';
	import LagList from './lag-list.svelte';
	import LanguageFormatMatrix from './language-format-matrix.svelte';
	import PlatformShareChart from './platform-share-chart.svelte';

	const { Story } = defineMeta({
		title: 'Spread/Sections',
		component: LagList,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { lags }
	});
</script>

<Story
	name="Public to WhatsApp lag"
	play={async ({ canvasElement }) => {
		const list = within(canvasElement).getByRole('list', {
			name: 'Public to WhatsApp lag by narrative'
		});
		const row = within(list).getByRole('link', { name: 'HPV vaccine causes sterility in girls' });
		await expect(row).toHaveAttribute('href', '/narratives/N-0361');
		// The 63 h label flips left of its marker so it stays inside the card.
		const label = within(list).getByText('+63 h');
		const card = canvasElement.querySelector('section')!.getBoundingClientRect();
		await expect(label.getBoundingClientRect().right).toBeLessThan(card.right);
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="Public → WhatsApp lag"
			description="Time from first public post to first WhatsApp question, by narrative"
			class="max-w-3xl"
		>
			<LagList {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Platform share over time"
	play={async ({ canvasElement }) => {
		const table = within(canvasElement).getByRole('table', {
			name: 'Share of incoming claims by platform, weekly'
		});
		await expect(within(table).getByRole('row', { name: /W34/ })).toHaveTextContent('52%');
	}}
>
	{#snippet template()}
		<SectionCard
			title="Platform share over time"
			description="Share of incoming claims by source, weekly"
			class="max-w-3xl"
		>
			<PlatformShareChart weeks={platformWeeks} />
			{#snippet footer()}<CoverageNote />{/snippet}
		</SectionCard>
	{/snippet}
</Story>

<Story name="Format of claims">
	{#snippet template()}
		<SectionCard
			title="Format of claims"
			description="Share by media type, weekly"
			class="max-w-sm"
		>
			<FormatShareChart weeks={formatWeeks} />
			<InsightCallout title="Voice notes now 52%." class="mt-3.5">
				Debunks reach more people as Mooré and Dioula audio than as French text.
			</InsightCallout>
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Language by format"
	play={async ({ canvasElement }) => {
		const table = within(canvasElement).getByRole('table');
		await expect(within(table).getByRole('row', { name: /Fulfulde/ })).toHaveTextContent('70%');
	}}
>
	{#snippet template()}
		<SectionCard
			title="Language × format"
			description="How each language arrives, last 30 d"
			class="max-w-sm"
		>
			<LanguageFormatMatrix rows={languageFormats} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Why people message"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getAllByRole('img')).toHaveLength(7);
		await expect(canvas.getByText('Out of scope')).toBeVisible();
	}}
>
	{#snippet template()}
		<SectionCard
			title="Why people message the bot"
			description="Intent of incoming WhatsApp messages, last 7 d"
			class="max-w-3xl"
		>
			<IntentBreakdown days={dailyIntents} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Empty states">
	{#snippet template()}
		<div class="grid max-w-3xl grid-cols-2 gap-5">
			<SectionCard title="Public → WhatsApp lag"><LagList lags={[]} /></SectionCard>
			<SectionCard title="Platform share"><PlatformShareChart weeks={[]} /></SectionCard>
			<SectionCard title="Language × format"><LanguageFormatMatrix rows={[]} /></SectionCard>
			<SectionCard title="Why people message"><IntentBreakdown days={[]} /></SectionCard>
		</div>
	{/snippet}
</Story>
