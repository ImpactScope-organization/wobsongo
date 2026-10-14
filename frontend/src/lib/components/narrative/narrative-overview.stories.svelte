<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import logo from '#lib/assets/wobsongo-mark.jpg';
	import {
		debunkFormats,
		engineVerdict,
		formats,
		languages,
		narrative,
		NARRATIVE_NOW,
		regionRespondents,
		regions,
		reviewLog,
		samples,
		spread
	} from '#lib/fixtures/narrative.js';
	import { formatTotal } from '#lib/utils/format.js';
	import AppHeader from '../common/app-header.svelte';
	import PageBreadcrumb from '../common/page-breadcrumb.svelte';
	import SectionCard from '../common/section-card.svelte';
	import AudienceBreakdown from './audience-breakdown.svelte';
	import DebunkFormats from './debunk-formats.svelte';
	import DismissMenu from './dismiss-menu.svelte';
	import NarrativeHeader from './narrative-header.svelte';
	import NarrativeMetrics from './narrative-metrics.svelte';
	import RegionList from '../common/region-list.svelte';
	import SampleMessages from './sample-messages.svelte';
	import SpreadTimeline from './spread-timeline.svelte';
	import VerdictPanel from './verdict-panel.svelte';

	// Composition only: how the narrative page will assemble these components,
	// for comparing against the design. The real page lives in routes/.
	const { Story } = defineMeta({
		title: 'Narrative/Overview',
		parameters: { layout: 'fullscreen' }
	});
</script>

<Story
	name="Narrative page"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent(narrative.title);
		await expect(canvas.getAllByRole('heading', { level: 2 })).toHaveLength(6);
	}}
>
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<AppHeader
				logoSrc={logo}
				currentHref="/narratives"
				items={[
					{ label: 'Pulse', href: '/pulse' },
					{ label: 'Narratives', href: '/narratives' },
					{ label: 'Spread', href: '/spread' },
					{ label: 'Operations', href: '/ops' },
					{ label: 'High-harm', href: '/safety', count: 3 }
				]}
			/>

			<main class="mx-auto flex max-w-[85rem] flex-col gap-5 p-[clamp(1rem,3vw,2rem)]">
				<div class="flex flex-col gap-3.5">
					<PageBreadcrumb
						crumbs={[
							{ label: 'Pulse', href: '/pulse' },
							{ label: 'Narratives' },
							{ label: narrative.id, mono: true }
						]}
					/>
					<NarrativeHeader {narrative} class="mb-0.5">
						{#snippet actions()}
							<DismissMenu narrativeId={narrative.id} action="?/dismiss" />
							<a
								href="/narratives/{narrative.id}/reply"
								class="rounded-md bg-primary px-3.5 py-[0.5625rem] text-[0.8125rem] font-semibold text-primary-foreground hover:bg-primary/85"
							>
								Edit reply
							</a>
						{/snippet}
					</NarrativeHeader>
				</div>

				<NarrativeMetrics {narrative} />

				<div class="flex flex-wrap items-start gap-5">
					<SectionCard
						title="How it spread"
						description="Mentions per hour by platform, last 72 h"
						class="flex-[2_1_35rem]"
					>
						<SpreadTimeline {spread} now={NARRATIVE_NOW} />
					</SectionCard>

					<SectionCard
						title="Verdict"
						description="Fact-check API result, open to human review"
						class="flex-[1_1_18.75rem]"
					>
						<VerdictPanel
							narrativeId={narrative.id}
							narrativeTitle={narrative.title}
							engine={engineVerdict}
							state={{ kind: 'pending' }}
							action="?/review"
							overrideHref="/narratives/{narrative.id}/reply"
							log={reviewLog}
							now={NARRATIVE_NOW}
						/>
					</SectionCard>

					<SectionCard title="Language" class="flex-[1_1_18.75rem]">
						<AudienceBreakdown {languages} {formats} />
					</SectionCard>

					<SectionCard
						title="Regions"
						description="Voluntary location answers in the bot, {formatTotal(regionRespondents)}"
						class="flex-[1_1_18.75rem]"
					>
						<RegionList {regions} />
					</SectionCard>

					<SectionCard
						title="What people are saying"
						description="Anonymized excerpts. Names, numbers and group names are removed."
						class="flex-[2_1_35rem]"
					>
						<SampleMessages {samples} />
					</SectionCard>

					<SectionCard
						title="Debunk formats"
						description="Suggested from language and format mix"
						class="flex-[1_1_18.75rem]"
					>
						<DebunkFormats formats={debunkFormats} />
					</SectionCard>
				</div>
			</main>
		</div>
	{/snippet}
</Story>
