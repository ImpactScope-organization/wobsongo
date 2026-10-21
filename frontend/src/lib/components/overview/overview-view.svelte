<script lang="ts">
	import {
		growthPct,
		misinfoShare,
		OVERVIEW_TOPICS,
		topCreators,
		trendDirection,
		type OverviewSnapshot,
		type OverviewTopic
	} from '#lib/domain/overview.js';
	import { TIME_RANGES, type TimeRange } from '#lib/domain/time-range.js';
	import { formatClockTime, formatNumber, formatPercent } from '#lib/utils/format.js';
	import AnomalyBanner from '../common/anomaly-banner.svelte';
	import CoverageNote from '../common/coverage-note.svelte';
	import KpiGroup from '../common/kpi-group.svelte';
	import KpiTile from '../common/kpi-tile.svelte';
	import { OVERVIEW_TOPIC_LABEL, TIME_RANGE_LABEL } from '../common/labels';
	import PageHeader from '../common/page-header.svelte';
	import SectionCard from '../common/section-card.svelte';
	import SegmentedLinks from '../common/segmented-links.svelte';
	import Sparkline from '../common/sparkline.svelte';
	import VerdictMeter from '../common/verdict-meter.svelte';
	import HighHarmSummary from './high-harm-summary.svelte';
	import NarrativeList from './narrative-list.svelte';
	import CreatorList from './creator-list.svelte';
	import PlatformBreakdown from './platform-breakdown.svelte';
	import { sampleCreators } from '#lib/fixtures/overview.js';

	interface Props {
		snapshot: OverviewSnapshot;
		range: TimeRange;
		topic: OverviewTopic;
		/** This page with another time range, e.g. "?range=7d&topic=all". */
		rangeHref: (range: TimeRange) => string;
		/** This page with another topic. */
		topicHref: (topic: OverviewTopic) => string;
		/** A narrative's detail page. */
		narrativeHref?: (id: string) => string;
		/** The high-harm view. */
		highHarmHref?: string;
		/** Whether the viewer has access to unredacted creator identities. */
		hasAccess?: boolean;
		/** Force privacy mode on/off. */
		privacyMode?: boolean;
	}

	let {
		snapshot,
		range,
		topic,
		rangeHref,
		topicHref,
		narrativeHref = (id) => `/narratives/${encodeURIComponent(id)}`,
		highHarmHref = '/safety',
		hasAccess = true,
		privacyMode = true
	}: Props = $props();

	const windowLabel = $derived(TIME_RANGE_LABEL[range]);

	/** "▲ 64%" style change; null when there is nothing to compare with. */
	function change(delta: number | null, text: (size: number) => string): string | undefined {
		if (delta === null) return undefined;
		if (Math.round(delta) === 0) return `No change vs prev. ${windowLabel}`;
		return `${delta > 0 ? '▲' : '▼'} ${text(Math.abs(delta))} vs prev. ${windowLabel}`;
	}

	const messageGrowth = $derived(growthPct(snapshot.messages));
	const debunkDelta = $derived(
		snapshot.medianDebunkMinutes.current - snapshot.medianDebunkMinutes.previous
	);
	const rated = $derived(misinfoShare(snapshot.rated));
	const visibleCreators = $derived(topCreators(snapshot.creators ?? sampleCreators, topic));
</script>

{#if snapshot.anomaly}
	<AnomalyBanner anomaly={snapshot.anomaly} href={narrativeHref(snapshot.anomaly.narrativeId)} />
{/if}

<main class="mx-auto flex max-w-[85rem] flex-col gap-5 p-[clamp(1rem,3vw,2rem)]">
	<PageHeader
		title="Overview"
		eyebrow="{snapshot.region} · updated {formatClockTime(snapshot.updatedAt)} GMT"
		class="mb-0.5"
	>
		{#snippet actions()}
			<SegmentedLinks
				label="Time range"
				current={range}
				options={TIME_RANGES.map((value) => ({
					value,
					label: TIME_RANGE_LABEL[value],
					href: rangeHref(value)
				}))}
			/>
		{/snippet}
	</PageHeader>

	<KpiGroup>
		<KpiTile
			label="Messages received, {windowLabel}"
			value={formatNumber(snapshot.messages.current)}
			caption={change(messageGrowth, (size) => formatPercent(size))}
			captionTone={messageGrowth !== null && messageGrowth > 0 ? 'signal' : 'muted'}
		>
			{#snippet aside()}
				<Sparkline
					values={snapshot.messages.trend}
					label="Messages over the last {windowLabel}, {trendDirection(snapshot.messages.trend)}"
					width={120}
					height={36}
					endDot
				/>
			{/snippet}
		</KpiTile>
		<KpiTile
			label="Active narratives"
			value={formatNumber(snapshot.activeNarratives.count)}
			caption="{formatNumber(snapshot.activeNarratives.new)} new in the last {windowLabel}"
		/>
		<KpiTile
			label="Rated false or misleading"
			value={formatNumber(rated.misinfo)}
			suffix="/ {formatNumber(rated.total)}"
		>
			<VerdictMeter counts={snapshot.rated} />
		</KpiTile>
		<KpiTile
			label="Median time to debunk"
			value={formatNumber(snapshot.medianDebunkMinutes.current)}
			suffix="min"
			caption={change(debunkDelta, (size) => `${formatNumber(size)} min`)}
			captionTone={debunkDelta > 0 ? 'signal' : 'muted'}
		/>
	</KpiGroup>

	<div class="flex flex-wrap items-start gap-5">
		<div class="grid min-w-0 flex-[2_1_35rem] gap-5">
			<SectionCard
				title={topic === 'misinfo' ? 'Top misinformation narratives' : 'Top narratives'}
				description="Clustered claims, last {windowLabel}, ranked by volume"
				flush
			>
				{#snippet actions()}
					<SegmentedLinks
						label="Topic"
						variant="pills"
						current={topic}
						options={OVERVIEW_TOPICS.map((value) => ({
							value,
							label: OVERVIEW_TOPIC_LABEL[value],
							href: topicHref(value)
						}))}
					/>
				{/snippet}
				<NarrativeList narratives={snapshot.narratives} hrefFor={narrativeHref} />
			</SectionCard>

			<SectionCard
				title="Content creators"
				description="Top content creators detected, last {windowLabel}, ranked by volume"
				flush
			>
				<CreatorList creators={visibleCreators} {hasAccess} {privacyMode} />
			</SectionCard>
		</div>

		<div class="flex min-w-0 flex-[1_1_20rem] flex-col gap-5">
			<SectionCard title="Where claims come from" description="Incoming claims by source platform">
				<PlatformBreakdown counts={snapshot.platformCounts} />
				{#snippet footer()}
					<CoverageNote />
				{/snippet}
			</SectionCard>
			<HighHarmSummary summary={snapshot.highHarm} href={highHarmHref} />
		</div>
	</div>
</main>
