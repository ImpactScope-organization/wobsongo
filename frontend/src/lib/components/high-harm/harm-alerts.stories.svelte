<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, userEvent, within } from 'storybook/test';
	import { harmAlerts, HIGH_HARM_NOW, substances } from '#lib/fixtures/high-harm.js';
	import SectionCard from '../common/section-card.svelte';
	import HarmAlertCard from './harm-alert-card.svelte';
	import HarmAlertList from './harm-alert-list.svelte';
	import SubstanceList from './substance-list.svelte';

	const { Story } = defineMeta({
		title: 'High-harm/Alerts',
		component: HarmAlertCard,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: {
			alert: harmAlerts[0],
			now: HIGH_HARM_NOW,
			action: '?/review',
			overrideHref: `/safety/${harmAlerts[0].id}/override`
		}
	});
</script>

<Story
	name="Critical, awaiting review"
	play={async ({ canvasElement }) => {
		const card = within(canvasElement).getByRole('article', {
			name: 'Potash (kanwa) + lemon · ingestion'
		});
		const form = card.querySelector('form')!;
		let posted: FormData | undefined;
		form.addEventListener('submit', (event) => {
			event.preventDefault();
			posted = new FormData(form, event.submitter);
		});

		await userEvent.click(within(card).getByRole('button', { name: 'Approve' }));
		await expect(posted?.get('id')).toBe('H-0091');
		await expect(posted?.get('decision')).toBe('approve');
		// Overriding needs a new verdict and reply, so it opens an editor page.
		await expect(within(card).getByRole('link', { name: 'Override…' })).toHaveAttribute(
			'href',
			'/safety/H-0091/override'
		);
	}}
>
	{#snippet template(args)}
		<HarmAlertCard {...args} class="max-w-2xl" />
	{/snippet}
</Story>

<Story name="High, reviewed" args={{ alert: harmAlerts[2] }}>
	{#snippet template(args)}
		<HarmAlertCard {...args} class="max-w-2xl" />
	{/snippet}
</Story>

<Story
	name="Dismissed"
	args={{ alert: { ...harmAlerts[1], review: { kind: 'decided', decision: 'dismiss' } } }}
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('status')).toHaveTextContent('Dismissed as irrelevant');
		await expect(canvas.getByRole('button', { name: 'Undo' })).toBeVisible();
	}}
>
	{#snippet template(args)}
		<HarmAlertCard {...args} class="max-w-2xl" />
	{/snippet}
</Story>

<Story
	name="Alert list"
	play={async ({ canvasElement }) => {
		const list = within(canvasElement).getByRole('list', { name: 'Dangerous practice alerts' });
		await expect(within(list).getAllByRole('article')).toHaveLength(3);
	}}
>
	{#snippet template()}
		<SectionCard
			title="Dangerous practice alerts"
			description="Claims promoting toxic substances or unsafe procedures. A medical expert should check every engine reply here."
			class="max-w-3xl"
		>
			<HarmAlertList
				alerts={harmAlerts}
				now={HIGH_HARM_NOW}
				action="?/review"
				overrideHrefFor={(id) => `/safety/${id}/override`}
			/>
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Substances"
	play={async ({ canvasElement }) => {
		const table = within(canvasElement).getByRole('table', {
			name: 'Substance and practice mentions'
		});
		await expect(within(table).getByRole('row', { name: /Bleach/ })).toHaveTextContent('9');
	}}
>
	{#snippet template()}
		<SectionCard
			title="Substances & practices mentioned"
			description="Matches against the clinical safety lexicon, last 7 d"
			class="max-w-sm"
		>
			<SubstanceList {substances} />
			{#snippet footer()}
				<p class="text-[0.71875rem]/[1.45] text-faint-foreground">
					Lexicon maintained by the medical advisory team.
				</p>
			{/snippet}
		</SectionCard>
	{/snippet}
</Story>

<Story name="Empty states">
	{#snippet template()}
		<div class="grid max-w-3xl grid-cols-2 gap-5">
			<SectionCard title="Dangerous practice alerts">
				<HarmAlertList
					alerts={[]}
					now={HIGH_HARM_NOW}
					action="?/review"
					overrideHrefFor={(id) => `/safety/${id}/override`}
				/>
			</SectionCard>
			<SectionCard title="Substances & practices mentioned">
				<SubstanceList substances={[]} />
			</SectionCard>
		</div>
	{/snippet}
</Story>
