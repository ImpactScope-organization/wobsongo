<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, userEvent, within } from 'storybook/test';
	import ReviewActions from './review-actions.svelte';

	/** Stops the real submit (it would navigate the story) and records what would be posted. */
	function captureSubmit(canvasElement: HTMLElement) {
		const form = canvasElement.querySelector('form')!;
		const posted: { data?: FormData } = {};
		form.addEventListener('submit', (event) => {
			event.preventDefault();
			posted.data = new FormData(form, event.submitter);
		});
		return posted;
	}

	const { Story } = defineMeta({
		title: 'Common/ReviewActions',
		component: ReviewActions,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: {
			itemId: 'c2',
			itemLabel: 'Contraceptive injections cause cancer',
			action: '?/review',
			overrideHref: '/ops/checks/c2/override',
			state: { kind: 'pending' }
		}
	});
</script>

<Story
	name="Pending"
	play={async ({ canvasElement }) => {
		const posted = captureSubmit(canvasElement);
		const canvas = within(canvasElement);
		await expect(
			canvas.getByRole('group', { name: 'Review “Contraceptive injections cause cancer”' })
		).toBeVisible();

		await userEvent.click(canvas.getByRole('button', { name: 'Approve' }));
		await expect(posted.data?.get('id')).toBe('c2');
		await expect(posted.data?.get('decision')).toBe('approve');
		await expect(canvas.getByRole('link', { name: 'Override…' })).toHaveAttribute(
			'href',
			'/ops/checks/c2/override'
		);
	}}
/>

<Story
	name="Knowledge gap"
	args={{ decisions: ['add-fact-check', 'dismiss'] }}
	play={async ({ canvasElement }) => {
		const posted = captureSubmit(canvasElement);
		await userEvent.click(within(canvasElement).getByRole('button', { name: 'Add fact-check' }));
		await expect(posted.data?.get('decision')).toBe('add-fact-check');
	}}
/>

<Story
	name="Approved"
	args={{ state: { kind: 'decided', decision: 'approve' } }}
	play={async ({ canvasElement }) => {
		const posted = captureSubmit(canvasElement);
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('status')).toHaveTextContent('Approved');
		await userEvent.click(canvas.getByRole('button', { name: 'Undo' }));
		await expect(posted.data?.get('decision')).toBe('undo');
	}}
/>

<Story name="Overridden" args={{ state: { kind: 'decided', decision: 'override' } }} />

<Story name="Dismissed" args={{ state: { kind: 'decided', decision: 'dismiss' } }} />
