<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, userEvent, within } from 'storybook/test';
	import { engineVerdict, narrative, NARRATIVE_NOW, reviewLog } from '#lib/fixtures/narrative.js';
	import SectionCard from '../common/section-card.svelte';
	import VerdictPanel from './verdict-panel.svelte';

	const { Story } = defineMeta({
		title: 'Narrative/VerdictPanel',
		component: VerdictPanel,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: {
			narrativeId: narrative.id,
			narrativeTitle: narrative.title,
			engine: engineVerdict,
			state: { kind: 'pending' },
			action: '?/review',
			overrideHref: `/narratives/${narrative.id}/reply`,
			log: reviewLog,
			now: NARRATIVE_NOW
		}
	});
</script>

<Story
	name="Awaiting review"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const form = canvasElement.querySelector('form')!;
		let posted: FormData | undefined;
		form.addEventListener('submit', (event) => {
			event.preventDefault();
			posted = new FormData(form, event.submitter);
		});

		await userEvent.click(canvas.getByRole('button', { name: 'Confirm verdict' }));
		await expect(posted?.get('decision')).toBe('approve');
		await expect(canvas.getAllByRole('listitem')).toHaveLength(3);
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="Verdict"
			description="Fact-check API result, open to human review"
			class="max-w-sm"
		>
			<VerdictPanel {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Confirmed" args={{ state: { kind: 'decided', decision: 'approve' } }}>
	{#snippet template(args)}
		<SectionCard title="Verdict" class="max-w-sm">
			<VerdictPanel {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Low-confidence misleading"
	args={{
		engine: {
			verdict: 'misleading',
			confidence: 0.58,
			explanation:
				'Partial match to “Injectable contraceptives and cancer risk” (match 0.61). Research shows small changes in risk for some cancers and lower risk for others.'
		},
		log: []
	}}
>
	{#snippet template(args)}
		<SectionCard title="Verdict" class="max-w-sm">
			<VerdictPanel {...args} />
		</SectionCard>
	{/snippet}
</Story>
