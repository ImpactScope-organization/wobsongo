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
	import { m } from '#lib/paraglide/messages.js';
	import { formatClockTime, formatNumber, formatPercent, plural } from '#lib/utils/format.js';
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
		if (Math.round(delta) === 0) return m.overview_no_change({ window: windowLabel });
		return m.overview_change({
			arrow: delta > 0 ? '▲' : '▼',
			size: text(Math.abs(delta)),
			window: windowLabel
		});
	}

	const TREND = {
		rising: m.overview_trend_rising,
		falling: m.overview_trend_falling,
		steady: m.overview_trend_steady
	};

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
		title={m.nav_overview()}
		eyebrow={m.overview_eyebrow({
			region: snapshot.region,
			time: formatClockTime(snapshot.updatedAt)
		})}
		class="mb-0.5"
	>
		{#snippet actions()}
			<SegmentedLinks
				label={m.overview_time_range()}
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
			label={m.overview_kpi_messages({ window: windowLabel })}
			value={formatNumber(snapshot.messages.current)}
			caption={change(messageGrowth, (size) => formatPercent(size))}
			captionTone={messageGrowth !== null && messageGrowth > 0 ? 'signal' : 'muted'}
		>
			{#snippet aside()}
				<Sparkline
					values={snapshot.messages.trend}
					label={m.overview_messages_trend({
						window: windowLabel,
						direction: TREND[trendDirection(snapshot.messages.trend)]()
					})}
					width={120}
					height={36}
					endDot
				/>
			{/snippet}
		</KpiTile>
		<KpiTile
			label={m.overview_kpi_active()}
			value={formatNumber(snapshot.activeNarratives.count)}
			caption={plural(
				snapshot.activeNarratives.new,
				m.overview_kpi_active_new_one,
				m.overview_kpi_active_new_other
			)({ count: formatNumber(snapshot.activeNarratives.new), window: windowLabel })}
		/>
		<KpiTile
			label={m.overview_kpi_rated()}
			value={formatNumber(rated.misinfo)}
			suffix="/ {formatNumber(rated.total)}"
		>
			<VerdictMeter counts={snapshot.rated} />
		</KpiTile>
		<KpiTile
			label={m.overview_kpi_debunk()}
			value={formatNumber(snapshot.medianDebunkMinutes.current)}
			suffix="min"
			caption={change(debunkDelta, (size) => `${formatNumber(size)} min`)}
			captionTone={debunkDelta > 0 ? 'signal' : 'muted'}
		/>
	</KpiGroup>

	<div class="flex flex-wrap items-start gap-5">
		<div class="grid min-w-0 flex-[2_1_35rem] gap-5">
			<SectionCard
				title={topic === 'misinfo' ? m.overview_top_misinfo() : m.overview_top()}
				description={m.overview_narratives_description({ window: windowLabel })}
				flush
			>
				{#snippet actions()}
					<SegmentedLinks
						label={m.overview_topic()}
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
				title={m.overview_creators_title()}
				description={m.overview_creators_description({ window: windowLabel })}
				flush
			>
				<CreatorList creators={visibleCreators} {hasAccess} {privacyMode} />
			</SectionCard>
		</div>

		<div class="flex min-w-0 flex-[1_1_20rem] flex-col gap-5">
			<SectionCard
				title={m.overview_sources_title()}
				description={m.overview_sources_description()}
			>
				<PlatformBreakdown counts={snapshot.platformCounts} />
				{#snippet footer()}
					<CoverageNote />
				{/snippet}
			</SectionCard>
			<HighHarmSummary summary={snapshot.highHarm} href={highHarmHref} />
		</div>
	</div>
</main>
