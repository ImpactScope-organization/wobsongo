<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import logo from '#lib/assets/ws-logo-white.svg';
	import {
		harmAlerts,
		HIGH_HARM_NOW,
		highHarmActivity,
		substances
	} from '#lib/fixtures/high-harm.js';
	import ActivityLog from '../common/activity-log.svelte';
	import AppHeader from '../common/app-header.svelte';
	import PageHeader from '../common/page-header.svelte';
	import SectionCard from '../common/section-card.svelte';
	import HarmAlertList from './harm-alert-list.svelte';
	import SubstanceList from './substance-list.svelte';

	// Composition only: how the high-harm page will assemble these components,
	// for comparing against the design. The real page lives in routes/.
	const { Story } = defineMeta({
		title: 'High-harm/Overview',
		parameters: { layout: 'fullscreen' }
	});
</script>

<Story
	name="High-harm page"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent('High-harm');
		await expect(canvas.getAllByRole('heading', { level: 2 })).toHaveLength(3);
		await expect(canvas.getAllByRole('heading', { level: 3 })).toHaveLength(3);
	}}
>
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<AppHeader
				logoSrc={logo}
				currentHref="/safety"
				items={[
					{ label: 'Overview', href: '/overview' },
					{ label: 'Narratives', href: '/narratives' },
					{ label: 'Spread', href: '/spread' },
					{ label: 'Operations', href: '/ops' },
					{ label: 'High-harm', href: '/safety', count: 3 }
				]}
			/>

			<main class="mx-auto flex max-w-[85rem] flex-col gap-5 p-[clamp(1rem,3vw,2rem)]">
				<PageHeader title="High-harm" eyebrow="Continuous stream filter" class="mb-0.5" />

				<div class="flex flex-wrap items-start gap-5">
					<SectionCard
						title="Dangerous practice alerts"
						description="Claims promoting toxic substances or unsafe procedures. A medical expert should check every engine reply here."
						class="flex-[2_1_35rem]"
					>
						<HarmAlertList
							alerts={harmAlerts}
							now={HIGH_HARM_NOW}
							action="?/review"
							overrideHrefFor={(id) => `/safety/${id}/override`}
						/>
					</SectionCard>

					<SectionCard
						title="Substances & practices mentioned"
						description="Matches against the clinical safety lexicon, last 7 d"
						class="flex-[1_1_20rem]"
					>
						<SubstanceList {substances} />
						{#snippet footer()}
							<p class="text-[0.71875rem]/[1.45] text-faint-foreground">
								Lexicon maintained by the medical advisory team.
							</p>
						{/snippet}
					</SectionCard>

					<SectionCard title="Activity" class="flex-[1_1_100%]">
						<ActivityLog entries={highHarmActivity} now={HIGH_HARM_NOW} />
					</SectionCard>
				</div>
			</main>
		</div>
	{/snippet}
</Story>
