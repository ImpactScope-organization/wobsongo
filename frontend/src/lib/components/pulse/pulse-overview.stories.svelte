<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { pulseSnapshot } from '#lib/fixtures/pulse.js';
	import logo from '#lib/assets/wobsongo-mark.jpg';
	import { NAV_ITEMS } from '#lib/navigation.js';
	import AppHeader from '../common/app-header.svelte';
	import PulseView from './pulse-view.svelte';

	// The Pulse page as routes/(app)/+page.svelte renders it, inside the app shell.
	const { Story } = defineMeta({
		title: 'Pulse/Overview',
		parameters: { layout: 'fullscreen' }
	});

	const hrefs = {
		rangeHref: (range: string) => `?range=${range}`,
		topicHref: (topic: string) => `?topic=${topic}`
	};
</script>

<Story
	name="Pulse page"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent('Pulse');
		await expect(canvas.getAllByRole('heading', { level: 2 })).toHaveLength(3);
		await expect(canvas.getByRole('link', { name: 'Pulse' })).toHaveAttribute(
			'aria-current',
			'page'
		);
		await expect(
			canvas.getByText('Narratives', { exact: false, selector: '[aria-disabled]' })
		).toHaveTextContent('coming soon');
	}}
>
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<AppHeader
				logoSrc={logo}
				currentHref="/"
				items={NAV_ITEMS}
				account={{ name: 'Dr. Aminata Ouédraogo', href: '/settings' }}
			/>
			<PulseView snapshot={pulseSnapshot} range="24h" topic="misinfo" {...hrefs} />
		</div>
	{/snippet}
</Story>

<Story name="Quiet week, all topics">
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<PulseView
				snapshot={{
					...pulseSnapshot,
					anomaly: null,
					messages: { ...pulseSnapshot.messages, previous: 9100 },
					medianDebunkMinutes: { current: 118, previous: 131 }
				}}
				range="7d"
				topic="all"
				{...hrefs}
			/>
		</div>
	{/snippet}
</Story>

<Story name="No narratives">
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<PulseView
				snapshot={{ ...pulseSnapshot, anomaly: null, narratives: [] }}
				range="24h"
				topic="misinfo"
				{...hrefs}
			/>
		</div>
	{/snippet}
</Story>
