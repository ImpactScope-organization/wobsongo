<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import logo from '#lib/assets/wobsongo-mark.jpg';
	import {
		debunkOutcomes,
		expertChecks,
		knowledgeGaps,
		OPERATIONS_NOW,
		priorityPoints
	} from '#lib/fixtures/operations.js';
	import AppHeader from '../common/app-header.svelte';
	import KpiGroup from '../common/kpi-group.svelte';
	import KpiTile from '../common/kpi-tile.svelte';
	import PageHeader from '../common/page-header.svelte';
	import SectionCard from '../common/section-card.svelte';
	import DebunkEffectiveness from './debunk-effectiveness.svelte';
	import ExpertCheckList from './expert-check-list.svelte';
	import KnowledgeGapList from './knowledge-gap-list.svelte';
	import PriorityMatrix from './priority-matrix.svelte';

	// Composition only: how the operations page will assemble these components,
	// for comparing against the design. The real page lives in routes/.
	const { Story } = defineMeta({
		title: 'Operations/Overview',
		parameters: { layout: 'fullscreen' }
	});
</script>

<Story
	name="Operations page"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent('Operations');
		await expect(canvas.getAllByRole('heading', { level: 2 })).toHaveLength(4);
		await expect(canvas.getAllByRole('term')).toHaveLength(4);
	}}
>
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<AppHeader
				logoSrc={logo}
				currentHref="/ops"
				items={[
					{ label: 'Pulse', href: '/pulse' },
					{ label: 'Narratives', href: '/narratives' },
					{ label: 'Spread', href: '/spread' },
					{ label: 'Operations', href: '/ops' },
					{ label: 'High-harm', href: '/safety', count: 3 }
				]}
			/>

			<main class="mx-auto flex max-w-[85rem] flex-col gap-5 p-[clamp(1rem,3vw,2rem)]">
				<PageHeader
					title="Operations"
					eyebrow="Engine results · updated in real time"
					class="mb-0.5"
				/>

				<KpiGroup>
					<KpiTile label="Net resolution rate" value="64%">
						<p class="text-xs/[1.4] text-muted-foreground">
							Users with no follow-up doubt within 24 h
						</p>
					</KpiTile>
					<KpiTile label="Positive feedback" value="71%">
						<p class="text-xs/[1.4] text-muted-foreground">“Was this helpful?” → Yes</p>
					</KpiTile>
					<KpiTile label="Secondary query rate" value="18%">
						<p class="text-xs/[1.4] text-muted-foreground">
							Asked again on the same topic within 24 h
						</p>
					</KpiTile>
					<KpiTile label="Expert overrides, 7 d" value="23">
						<p class="text-xs/[1.4] text-muted-foreground">
							4% of engine results changed by an expert
						</p>
					</KpiTile>
				</KpiGroup>

				<div class="flex flex-wrap items-start gap-5">
					<SectionCard
						title="Needs expert check"
						description="Engine results flagged for low confidence, high harm or partial truth. Approve, override or dismiss."
						class="flex-[1_1_100%]"
						flush
					>
						<ExpertCheckList
							checks={expertChecks}
							action="?/review"
							overrideHrefFor={(id) => `/ops/checks/${id}/override`}
						/>
					</SectionCard>

					<SectionCard
						title="Harm × spread"
						description="Active narratives. Bubble size = 7-day volume"
						class="flex-[1_1_22.5rem]"
					>
						<PriorityMatrix points={priorityPoints} />
					</SectionCard>

					<SectionCard
						title="Not in the knowledge base"
						description="Claims the engine could not match (score < 0.65), clustered, last 7 d"
						class="flex-[2_1_35rem]"
					>
						<KnowledgeGapList gaps={knowledgeGaps} now={OPERATIONS_NOW} action="?/gap" />
					</SectionCard>

					<SectionCard
						title="Debunk effectiveness"
						description="WhatsApp reactions to delivered debunks, last 30 d"
						class="flex-[1_1_100%]"
					>
						<DebunkEffectiveness outcomes={debunkOutcomes} />
					</SectionCard>
				</div>
			</main>
		</div>
	{/snippet}
</Story>
