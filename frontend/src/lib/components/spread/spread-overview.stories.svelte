<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import logo from '#lib/assets/ws-logo-white.svg';
	import {
		allRegionRespondents,
		allRegions,
		dailyIntents,
		formatWeeks,
		languageFormats,
		lags,
		platformWeeks
	} from '#lib/fixtures/spread.js';
	import { formatTotal } from '#lib/utils/format.js';
	import AppHeader from '../common/app-header.svelte';
	import CoverageNote from '../common/coverage-note.svelte';
	import InsightCallout from '../common/insight-callout.svelte';
	import PageHeader from '../common/page-header.svelte';
	import RegionList from '../common/region-list.svelte';
	import SectionCard from '../common/section-card.svelte';
	import SegmentedLinks from '../common/segmented-links.svelte';
	import FormatShareChart from './format-share-chart.svelte';
	import IntentBreakdown from './intent-breakdown.svelte';
	import LagList from './lag-list.svelte';
	import LanguageFormatMatrix from './language-format-matrix.svelte';
	import PlatformShareChart from './platform-share-chart.svelte';

	// Composition only: how the spread page will assemble these components,
	// for comparing against the design. The real page lives in routes/.
	const { Story } = defineMeta({
		title: 'Spread/Overview',
		parameters: { layout: 'fullscreen' }
	});
</script>

<Story
	name="Spread page"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent('Spread');
		await expect(canvas.getAllByRole('heading', { level: 2 })).toHaveLength(6);
	}}
>
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<AppHeader
				logoSrc={logo}
				currentHref="/spread"
				items={[
					{ label: 'Overview', href: '/overview' },
					{ label: 'Narratives', href: '/narratives' },
					{ label: 'Spread', href: '/spread' },
					{ label: 'Operations', href: '/ops' },
					{ label: 'High-harm', href: '/safety', count: 3 }
				]}
			/>

			<main class="mx-auto flex max-w-[85rem] flex-col gap-5 p-[clamp(1rem,3vw,2rem)]">
				<PageHeader title="Spread" eyebrow="Daily batch · last run 00:00 GMT" class="mb-0.5">
					{#snippet actions()}
						<SegmentedLinks
							label="Time range"
							current="8w"
							options={[
								{ value: '7d', label: '7 d', href: '?range=7d' },
								{ value: '8w', label: '8 wk', href: '?range=8w' },
								{ value: '6m', label: '6 mo', href: '?range=6m' }
							]}
						/>
					{/snippet}
				</PageHeader>

				<div class="flex flex-wrap items-start gap-5">
					<SectionCard
						title="Platform share over time"
						description="Share of incoming claims by source, weekly"
						class="flex-[2_1_35rem]"
					>
						<PlatformShareChart weeks={platformWeeks} />
						{#snippet footer()}<CoverageNote />{/snippet}
					</SectionCard>

					<SectionCard
						title="Format of claims"
						description="Share by media type, weekly"
						class="flex-[1_1_20rem]"
					>
						<FormatShareChart weeks={formatWeeks} />
						<InsightCallout title="Voice notes now 52%." class="mt-3.5">
							Debunks reach more people as Mooré and Dioula audio than as French text.
						</InsightCallout>
					</SectionCard>

					<SectionCard
						title="Public → WhatsApp lag"
						description="Time from first public post to first WhatsApp question, by narrative"
						class="flex-[2_1_35rem]"
					>
						<LagList {lags} />
					</SectionCard>

					<SectionCard
						title="Language × format"
						description="How each language arrives, last 30 d"
						class="flex-[1_1_20rem]"
					>
						<LanguageFormatMatrix rows={languageFormats} />
					</SectionCard>

					<SectionCard
						title="Why people message the bot"
						description="Intent of incoming WhatsApp messages, last 7 d"
						class="flex-[2_1_35rem]"
					>
						<IntentBreakdown days={dailyIntents} />
					</SectionCard>

					<SectionCard
						title="Regions"
						description="Voluntary location answers in the bot, last 30 d, {formatTotal(
							allRegionRespondents
						)}"
						class="flex-[1_1_20rem]"
					>
						<RegionList regions={allRegions} />
					</SectionCard>
				</div>
			</main>
		</div>
	{/snippet}
</Story>
