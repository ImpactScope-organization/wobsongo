<script lang="ts">
	import {
		ENGAGEMENT_WINDOWS,
		type EngagementWindow,
		type UserQuestionStats
	} from '#lib/domain/overview.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatNumber } from '#lib/utils/format.js';
	import KpiTile from '../common/kpi-tile.svelte';
	import { ENGAGEMENT_WINDOW_LABEL, ENGAGEMENT_WINDOW_LONG_LABEL } from '../common/labels';
	import SectionCard from '../common/section-card.svelte';
	import SegmentedLinks from '../common/segmented-links.svelte';

	interface Props {
		// Unique users and questions
		stats: UserQuestionStats;
		engagement: EngagementWindow;
		// This page with another engagement window, e.g. "?engagement=7d".
		engagementHref: (engagement: EngagementWindow) => string;
		class?: string;
	}

	let { stats, engagement, engagementHref, class: className }: Props = $props();
</script>

<SectionCard
	title={m.overview_user_questions_title()}
	description={m.overview_user_questions_description({
		window: ENGAGEMENT_WINDOW_LONG_LABEL[engagement]
	})}
	flush
	class={className}
>
	{#snippet actions()}
		<SegmentedLinks
			label={m.overview_time_range()}
			variant="pills"
			current={engagement}
			options={ENGAGEMENT_WINDOWS.map((value) => ({
				value,
				label: ENGAGEMENT_WINDOW_LABEL[value],
				href: engagementHref(value)
			}))}
		/>
	{/snippet}
	<dl
		role="group"
		aria-label={m.overview_user_questions_title()}
		class="grid grid-cols-2 gap-px border-t bg-border"
	>
		<KpiTile size="sm" label={m.overview_kpi_unique_users()} value={formatNumber(stats.users)} />
		<KpiTile size="sm" label={m.overview_kpi_questions()} value={formatNumber(stats.questions)} />
	</dl>
</SectionCard>
