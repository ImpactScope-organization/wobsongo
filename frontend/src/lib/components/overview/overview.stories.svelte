<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import { overviewSnapshot } from '#lib/fixtures/overview.js';
	import logo from '#lib/assets/ws-logo-white.svg';
	import { NAV_ITEMS } from '#lib/navigation.js';
	import AppHeader from '../common/app-header.svelte';
	import OverviewView from './overview-view.svelte';

	// The Overview page as routes/(app)/+page.svelte renders it, inside the app shell.
	const { Story } = defineMeta({
		title: 'Overview/Overview',
		parameters: { layout: 'fullscreen' }
	});

	const hrefs = {
		rangeHref: (range: string) => `?range=${range}`,
		topicHref: (topic: string) => `?topic=${topic}`
	};
</script>

<Story
	name="Overview page"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent('Overview');
		await expect(canvas.getAllByRole('heading', { level: 2 })).toHaveLength(4);
		await expect(
			canvas.getByRole('heading', { level: 2, name: 'Content creators' })
		).toBeInTheDocument();
		await expect(canvas.getByRole('link', { name: 'Overview' })).toHaveAttribute(
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
			<OverviewView snapshot={overviewSnapshot} range="24h" topic="misinfo" {...hrefs} />
		</div>
	{/snippet}
</Story>

<Story name="Quiet week, all topics">
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<OverviewView
				snapshot={{
					...overviewSnapshot,
					anomaly: null,
					messages: { ...overviewSnapshot.messages, previous: 9100 },
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
			<OverviewView
				snapshot={{ ...overviewSnapshot, anomaly: null, narratives: [] }}
				range="24h"
				topic="misinfo"
				{...hrefs}
			/>
		</div>
	{/snippet}
</Story>
