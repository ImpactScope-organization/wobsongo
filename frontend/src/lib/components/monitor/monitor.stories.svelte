<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import {
		monitorSnapshot,
		monitorWindows,
		processingRuns,
		runningRun
	} from '#lib/fixtures/monitor.js';
	import logo from '#lib/assets/ws-logo-white.svg';
	import { navItems } from '#lib/navigation.js';
	import AppHeader from '../common/app-header.svelte';
	import MonitorView from './monitor-view.svelte';

	// The Monitor page as routes/(app)/monitor/+page.svelte renders it, inside the app shell.
	const { Story } = defineMeta({
		title: 'Monitor/Monitor',
		parameters: { layout: 'fullscreen' }
	});

	const hrefs = {
		rangeHref: (range: string) => `?range=${range}`,
		runHref: (id: string) => `?run=${id}`
	};
</script>

<Story
	name="Monitor page"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent('Monitor');
		await expect(canvas.getAllByRole('heading', { level: 2 })).toHaveLength(4);
		const nav = within(canvas.getByRole('navigation', { name: 'Main' }));
		await expect(nav.getByRole('link', { name: 'Monitor' })).toHaveAttribute(
			'aria-current',
			'page'
		);
		await expect(canvas.getByRole('button', { name: 'Run now' })).toBeEnabled();
	}}
>
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<AppHeader
				logoSrc={logo}
				currentHref="/monitor"
				items={navItems()}
				account={{ name: 'Dr. Aminata Ouédraogo', href: '/settings' }}
			/>
			<MonitorView snapshot={monitorSnapshot} range="24h" {...hrefs} />
		</div>
	{/snippet}
</Story>

<Story name="Manual run selected, 7 days">
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<MonitorView
				snapshot={{ ...monitorSnapshot, ...monitorWindows['7d'] }}
				range="7d"
				selectedRunId={processingRuns[2].id}
				{...hrefs}
			/>
		</div>
	{/snippet}
</Story>

<Story
	name="Run in progress"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		// Still focusable, so screen readers find it; it just doesn't submit.
		const button = canvas.getByRole('button', { name: 'Running…' });
		await expect(button).toHaveAttribute('aria-disabled', 'true');
		button.focus();
		await expect(button).toHaveFocus();
		await expect(canvas.getByText('Run in progress…')).toBeVisible();
	}}
>
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<MonitorView
				snapshot={{ ...monitorSnapshot, runs: [runningRun, ...processingRuns] }}
				range="24h"
				runError="A run is already in progress. Wait for it to finish."
				{...hrefs}
			/>
		</div>
	{/snippet}
</Story>

<Story name="No runs yet">
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<MonitorView
				snapshot={{
					...monitorSnapshot,
					runs: [],
					runCount: 0,
					costUsd: 0,
					volume: { bucketHours: 2, buckets: [] }
				}}
				range="24h"
				{...hrefs}
			/>
		</div>
	{/snippet}
</Story>
