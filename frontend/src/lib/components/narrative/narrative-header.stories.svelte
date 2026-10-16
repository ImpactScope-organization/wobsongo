<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, userEvent, within } from 'storybook/test';
	import { narrative } from '#lib/fixtures/narrative.js';
	import PageBreadcrumb from '../common/page-breadcrumb.svelte';
	import DismissMenu from './dismiss-menu.svelte';
	import NarrativeHeader from './narrative-header.svelte';

	/** Stops the real submit (it would navigate the story) and records what would be posted. */
	function captureSubmit(form: HTMLFormElement) {
		const posted: { data?: FormData } = {};
		form.addEventListener('submit', (event) => {
			event.preventDefault();
			posted.data = new FormData(form, event.submitter);
		});
		return posted;
	}

	const { Story } = defineMeta({
		title: 'Narrative/NarrativeHeader',
		component: NarrativeHeader,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { narrative }
	});
</script>

<Story
	name="With breadcrumb and actions"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent(narrative.title);
		await expect(canvas.getByRole('link', { name: 'Overview' })).toHaveAttribute(
			'href',
			'/overview'
		);

		// The dismiss menu is a <details>: closed until its summary is clicked.
		const reason = () => canvas.queryByRole('button', { name: /Duplicate/ });
		await expect(reason()).not.toBeVisible();
		await userEvent.click(canvas.getByText('Dismiss'));
		await expect(reason()).toBeVisible();

		const posted = captureSubmit(canvasElement.querySelector('details form')!);
		await userEvent.click(reason()!);
		await expect(posted.data?.get('id')).toBe('N-0412');
		await expect(posted.data?.get('reason')).toBe('duplicate');
	}}
>
	{#snippet template(args)}
		<div class="flex flex-col gap-3.5">
			<PageBreadcrumb
				crumbs={[
					{ label: 'Overview', href: '/overview' },
					{ label: 'Narratives' },
					{ label: narrative.id, mono: true }
				]}
			/>
			<NarrativeHeader {...args}>
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
	{/snippet}
</Story>

<Story
	name="Dismiss menu open"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getAllByRole('button')).toHaveLength(4);
		await userEvent.keyboard('{Escape}');
		await expect(canvas.queryByRole('button', { name: /Duplicate/ })).not.toBeVisible();
	}}
>
	{#snippet template()}
		<div class="flex h-80 justify-end">
			<DismissMenu narrativeId={narrative.id} action="?/dismiss" open />
		</div>
	{/snippet}
</Story>
