<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import {
		anomaly,
		highHarmSummary,
		messagesTrend,
		narratives,
		platformCounts,
		PULSE_NOW,
		ratedBreakdown
	} from '#lib/fixtures/pulse.js';
	import logo from '#lib/assets/wobsongo-mark.jpg';
	import { formatClockTime } from '#lib/utils/format.js';
	import AnomalyBanner from '../common/anomaly-banner.svelte';
	import AppHeader from '../common/app-header.svelte';
	import CoverageNote from '../common/coverage-note.svelte';
	import KpiGroup from '../common/kpi-group.svelte';
	import KpiTile from '../common/kpi-tile.svelte';
	import PageHeader from '../common/page-header.svelte';
	import SectionCard from '../common/section-card.svelte';
	import SegmentedLinks from '../common/segmented-links.svelte';
	import Sparkline from '../common/sparkline.svelte';
	import VerdictMeter from '../common/verdict-meter.svelte';
	import HighHarmSummary from './high-harm-summary.svelte';
	import NarrativeList from './narrative-list.svelte';
	import PlatformBreakdown from './platform-breakdown.svelte';

	// Composition only: how the Pulse page will assemble these components,
	// for comparing against the design. The real page lives in routes/.
	const { Story } = defineMeta({
		title: 'Pulse/Overview',
		parameters: { layout: 'fullscreen' }
	});
</script>

<Story
	name="Pulse page"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent('Pulse');
		await expect(canvas.getAllByRole('heading', { level: 2 })).toHaveLength(3);
	}}
>
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<AppHeader
				logoSrc={logo}
				currentHref="/pulse"
				items={[
					{ label: 'Pulse', href: '/pulse' },
					{ label: 'Narratives', href: '/narratives' },
					{ label: 'Spread', href: '/spread' },
					{ label: 'Operations', href: '/ops' },
					{ label: 'High-harm', href: '/safety', count: 3 }
				]}
			/>
			<AnomalyBanner {anomaly} href="/narratives/{anomaly.narrativeId}" />

			<main class="mx-auto flex max-w-[85rem] flex-col gap-5 p-[clamp(1rem,3vw,2rem)]">
				<PageHeader
					title="Pulse"
					eyebrow="Burkina Faso · updated {formatClockTime(PULSE_NOW)} GMT"
					class="mb-0.5"
				>
					{#snippet actions()}
						<SegmentedLinks
							label="Time range"
							current="24h"
							options={[
								{ value: '24h', label: '24 h', href: '?range=24h' },
								{ value: '7d', label: '7 d', href: '?range=7d' },
								{ value: '30d', label: '30 d', href: '?range=30d' }
							]}
						/>
					{/snippet}
				</PageHeader>

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

				<div class="flex flex-wrap items-start gap-5">
					<SectionCard
						title="Top misinformation narratives"
						description="Clustered claims, last 24 h, ranked by volume"
						flush
						class="flex-[2_1_35rem]"
					>
						{#snippet actions()}
							<SegmentedLinks
								label="Topic"
								variant="pills"
								current="misinfo"
								options={[
									{ value: 'misinfo', label: 'Misinformation', href: '?topic=misinfo' },
									{ value: 'all', label: 'All topics', href: '?topic=all' }
								]}
							/>
						{/snippet}
						<NarrativeList {narratives} />
					</SectionCard>

					<div class="flex min-w-0 flex-[1_1_20rem] flex-col gap-5">
						<SectionCard
							title="Where claims come from"
							description="Incoming claims by source platform"
						>
							<PlatformBreakdown counts={platformCounts} />
							{#snippet footer()}
								<CoverageNote />
							{/snippet}
						</SectionCard>
						<HighHarmSummary summary={highHarmSummary} href="/safety" />
					</div>
				</div>
			</main>
		</div>
	{/snippet}
</Story>
