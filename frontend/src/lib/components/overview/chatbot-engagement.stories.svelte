<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, userEvent, within } from 'storybook/test';
	import { sampleUserQuestions } from '#lib/fixtures/overview.js';
	import ChatbotEngagement from './chatbot-engagement.svelte';

	const { Story } = defineMeta({
		title: 'Overview/ChatbotEngagement',
		component: ChatbotEngagement,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { stats: sampleUserQuestions, class: 'max-w-md' }
	});
</script>

<Story
	name="Default (Last 7 days)"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(
			canvas.getByRole('heading', { level: 2, name: 'Chatbot engagement' })
		).toBeVisible();
		await expect(canvas.getByText('Total unique users and questions · Last 7 days')).toBeVisible();
		await expect(canvas.getByRole('button', { name: '7 d' })).toHaveAttribute(
			'aria-pressed',
			'true'
		);
		await expect(canvas.getByText('2')).toBeVisible();
		await expect(canvas.getByText('24')).toBeVisible();

		// Switch through the time filters interactively
		await userEvent.click(canvas.getByRole('button', { name: '15 d' }));
		await expect(canvas.getByText('Total unique users and questions · Last 15 days')).toBeVisible();
		await expect(canvas.getByText('4')).toBeVisible();
		await expect(canvas.getByText('33')).toBeVisible();

		await userEvent.click(canvas.getByRole('button', { name: '30 d' }));
		await expect(canvas.getByText('Total unique users and questions · Last 30 days')).toBeVisible();
		await expect(canvas.getByText('14')).toBeVisible();
		await expect(canvas.getByText('159')).toBeVisible();

		await userEvent.click(canvas.getByRole('button', { name: 'All time' }));
		await expect(canvas.getByText('Total unique users and questions · All time')).toBeVisible();
		await expect(canvas.getByText('59')).toBeVisible();
		await expect(canvas.getByText('502')).toBeVisible();
	}}
/>

<Story name="Last 15 days" args={{ defaultWindow: '15d' }} />

<Story name="Last 30 days" args={{ defaultWindow: '30d' }} />

<Story name="All time" args={{ defaultWindow: 'all' }} />
