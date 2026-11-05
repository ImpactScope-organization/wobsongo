<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { sampleUserQuestions } from '#lib/fixtures/overview.js';
	import ChatbotEngagement from './chatbot-engagement.svelte';
	
	// The Overview page as routes/(app)/+page.svelte renders it, inside the app shell.
	const { Story } = defineMeta({
		title: 'Overview/ChatbotEngagement',
		component: ChatbotEngagement,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: {
			stats: sampleUserQuestions.all,
			engagement: 'all',
			engagementHref: (engagement: string) => `?engagement=${engagement}`,
			class: 'max-w-md'
		}
	});
</script>

<Story
	name="Default (All time)"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(
			canvas.getByRole('heading', { level: 2, name: 'Chatbot engagement' })
		).toBeVisible();
		await expect(canvas.getByText('Total unique users and questions · All time')).toBeVisible();
		await expect(canvas.getByRole('link', { name: 'All time' })).toHaveAttribute(
			'aria-current',
			'true'
		);
		await expect(canvas.getByRole('link', { name: '7 d' })).toHaveAttribute(
			'href',
			'?engagement=7d'
		);
		await expect(canvas.getByText('59')).toBeVisible();
		await expect(canvas.getByText('502')).toBeVisible();
	}}
/>

<Story
	name="Last 7 days"
	args={{ engagement: '7d', stats: sampleUserQuestions['7d'] }}
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('Total unique users and questions · Last 7 days')).toBeVisible();
		await expect(canvas.getByRole('link', { name: '7 d' })).toHaveAttribute('aria-current', 'true');
		await expect(canvas.getByText('2')).toBeVisible();
		await expect(canvas.getByText('24')).toBeVisible();
	}}
/>

<Story name="Last 15 days" args={{ engagement: '15d', stats: sampleUserQuestions['15d'] }} />

<Story name="Last 30 days" args={{ engagement: '30d', stats: sampleUserQuestions['30d'] }} />
