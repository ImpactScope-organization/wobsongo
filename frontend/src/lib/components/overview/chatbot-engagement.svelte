<script lang="ts">
	import {
		ENGAGEMENT_WINDOWS,
		type EngagementWindow,
		type UserQuestionStats
	} from '#lib/domain/overview.js';
	import type { TimeRange } from '#lib/domain/time-range.js';
	import { sampleUserQuestions } from '#lib/fixtures/overview.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatNumber } from '#lib/utils/format.js';
	import KpiTile from '../common/kpi-tile.svelte';
	import { ENGAGEMENT_WINDOW_LABEL, ENGAGEMENT_WINDOW_LONG_LABEL } from '../common/labels';
	import SectionCard from '../common/section-card.svelte';

	interface Props {
		stats?: Record<EngagementWindow, UserQuestionStats>;
		/** Initial window or page-level time range to sync with until a pill is clicked. */
		defaultWindow?: EngagementWindow | TimeRange;
		class?: string;
	}

	let { stats = sampleUserQuestions, defaultWindow = '7d', class: className }: Props = $props();

	let selectedWindow = $state<EngagementWindow | null>(null);
	const activeWindow = $derived<EngagementWindow>(
		selectedWindow ??
			(defaultWindow === '15d' || defaultWindow === '30d' || defaultWindow === 'all'
				? defaultWindow
				: '7d')
	);
	const current = $derived(stats[activeWindow]);
</script>

<SectionCard
	title={m.overview_user_questions_title()}
	description={m.overview_user_questions_description({
		window: ENGAGEMENT_WINDOW_LONG_LABEL[activeWindow]
	})}
	flush
	class={className}
>
	{#snippet actions()}
		<nav aria-label={m.overview_time_range()}>
			<ul class="flex flex-wrap gap-1.5">
				{#each ENGAGEMENT_WINDOWS as value (value)}
					<li>
						<button
							type="button"
							aria-pressed={value === activeWindow}
							onclick={() => (selectedWindow = value)}
							class="block cursor-pointer rounded-full border px-2.5 py-1 text-[0.6875rem] font-medium whitespace-nowrap text-muted-foreground transition-colors outline-none hover:border-foreground hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring aria-pressed:border-primary aria-pressed:bg-primary aria-pressed:text-primary-foreground"
						>
							{ENGAGEMENT_WINDOW_LABEL[value]}
						</button>
					</li>
				{/each}
			</ul>
		</nav>
	{/snippet}
	<dl
		role="group"
		aria-label={m.overview_user_questions_title()}
		class="grid grid-cols-2 gap-px border-t bg-border"
	>
		<KpiTile size="sm" label={m.overview_kpi_unique_users()} value={formatNumber(current.users)} />
		<KpiTile size="sm" label={m.overview_kpi_questions()} value={formatNumber(current.questions)} />
	</dl>
</SectionCard>
