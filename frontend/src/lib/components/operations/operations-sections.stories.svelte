<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, userEvent, within } from 'storybook/test';
	import {
		debunkOutcomes,
		expertChecks,
		knowledgeGaps,
		OPERATIONS_NOW,
		priorityPoints
	} from '#lib/fixtures/operations.js';
	import SectionCard from '../common/section-card.svelte';
	import DebunkEffectiveness from './debunk-effectiveness.svelte';
	import ExpertCheckList from './expert-check-list.svelte';
	import KnowledgeGapList from './knowledge-gap-list.svelte';
	import PriorityMatrix from './priority-matrix.svelte';

	const { Story } = defineMeta({
		title: 'Operations/Sections',
		component: ExpertCheckList,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: {
			checks: expertChecks,
			action: '?/review',
			overrideHrefFor: (id: string) => `/ops/checks/${id}/override`
		}
	});

	function captureSubmit(form: HTMLFormElement) {
		const posted: { data?: FormData } = {};
		form.addEventListener('submit', (event) => {
			event.preventDefault();
			posted.data = new FormData(form, event.submitter);
		});
		return posted;
	}
</script>

<Story
	name="Needs expert check"
	play={async ({ canvasElement }) => {
		const list = within(canvasElement).getByRole('list', { name: 'Expert review queue' });
		const forms = list.querySelectorAll('form');
		await expect(forms).toHaveLength(5);
		const posted = captureSubmit(forms[1]);
		await userEvent.click(within(forms[1]).getByRole('button', { name: 'Approve' }));
		await expect(posted.data?.get('id')).toBe('C-0202');
		await expect(posted.data?.get('decision')).toBe('approve');
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="Needs expert check"
			description="Engine results flagged for low confidence, high harm or partial truth. Approve, override or dismiss."
			class="max-w-5xl"
			flush
		>
			<ExpertCheckList {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Harm × spread"
	exportName="HarmBySpread"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const chart = canvas.getByRole('img', { name: /Urgent debunk:/ });
		// Scales with the card instead of overflowing it.
		const svg = chart.querySelector('svg')!.getBoundingClientRect();
		await expect(Math.abs(svg.width - chart.getBoundingClientRect().width)).toBeLessThan(1);
		await expect(
			canvas.getByRole('link', { name: /Implants cause permanent infertility/ })
		).toBeVisible();
	}}
>
	{#snippet template()}
		<SectionCard
			title="Harm × spread"
			description="Active narratives. Bubble size = 7-day volume"
			class="max-w-md"
		>
			<PriorityMatrix points={priorityPoints} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Knowledge gaps"
	play={async ({ canvasElement }) => {
		const list = within(canvasElement).getByRole('list', { name: 'Knowledge base gaps' });
		const form = list.querySelector('form')!;
		const posted = captureSubmit(form);
		await userEvent.click(within(form).getByRole('button', { name: 'Add fact-check' }));
		await expect(posted.data?.get('decision')).toBe('add-fact-check');
	}}
>
	{#snippet template()}
		<SectionCard
			title="Not in the knowledge base"
			description="Claims the engine could not match (score < 0.65), clustered, last 7 d"
			class="max-w-3xl"
		>
			<KnowledgeGapList gaps={knowledgeGaps} now={OPERATIONS_NOW} action="?/gap" />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Debunk effectiveness"
	play={async ({ canvasElement }) => {
		const table = within(canvasElement).getByRole('table', { name: 'Debunk effectiveness' });
		await expect(within(table).getByRole('row', { name: /Condom myths/ })).toHaveTextContent('31%');
	}}
>
	{#snippet template()}
		<SectionCard
			title="Debunk effectiveness"
			description="WhatsApp reactions to delivered debunks, last 30 d"
			class="max-w-5xl"
		>
			<DebunkEffectiveness outcomes={debunkOutcomes} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Empty states">
	{#snippet template()}
		<div class="grid max-w-3xl grid-cols-2 gap-5">
			<SectionCard title="Needs expert check" flush>
				<ExpertCheckList checks={[]} action="?/review" overrideHrefFor={() => ''} />
			</SectionCard>
			<SectionCard title="Harm × spread"><PriorityMatrix points={[]} /></SectionCard>
			<SectionCard title="Not in the knowledge base">
				<KnowledgeGapList gaps={[]} now={OPERATIONS_NOW} action="?/gap" />
			</SectionCard>
			<SectionCard title="Debunk effectiveness"><DebunkEffectiveness outcomes={[]} /></SectionCard>
		</div>
	{/snippet}
</Story>
