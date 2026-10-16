<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, waitFor, within } from 'storybook/test';
	import logo from '#lib/assets/ws-logo-white.svg';
	import { anomaly } from '#lib/fixtures/overview.js';
	import AnomalyBanner from './anomaly-banner.svelte';
	import AppHeader from './app-header.svelte';
	import PageHeader from './page-header.svelte';
	import SegmentedLinks from './segmented-links.svelte';

	const NAV = [
		{ label: 'Overview', href: '/overview' },
		{ label: 'Narratives', href: '/narratives' },
		{ label: 'Spread', href: '/spread' },
		{ label: 'Operations', href: '/ops' },
		{ label: 'High-harm', href: '/safety', count: 3 }
	];

	const RANGES = [
		{ value: '24h', label: '24 h', href: '?range=24h' },
		{ value: '7d', label: '7 d', href: '?range=7d' },
		{ value: '30d', label: '30 d', href: '?range=30d' }
	];

	const { Story } = defineMeta({
		title: 'Common/Navigation',
		component: AppHeader,
		tags: ['autodocs'],
		parameters: { layout: 'fullscreen' },
		args: { items: NAV, currentHref: '/overview', logoSrc: logo }
	});
</script>

<Story
	name="App header"
	play={async ({ canvasElement }) => {
		const nav = within(canvasElement).getByRole('navigation', { name: 'Main' });
		const current = within(nav).getByRole('link', { current: 'page' });
		await expect(current).toHaveTextContent('Overview');
		const brand = within(canvasElement).getByRole('link', { name: 'Monitor' });
		const logoImg = brand.querySelector('img')!;
		// Proves the asset path resolves: a broken image has no natural size.
		await waitFor(() => expect(logoImg.naturalWidth).toBeGreaterThan(0));
		await expect(within(nav).getByRole('link', { name: /High-harm 3 open/ })).toHaveAttribute(
			'href',
			'/safety'
		);
	}}
>
	{#snippet template(args)}
		<AppHeader {...args}>
			{#snippet actions()}
				<a
					href="/digest.pdf"
					class="rounded-md bg-inverse-foreground px-3 py-[0.4375rem] text-xs font-semibold text-inverse hover:bg-white"
				>
					Export digest
				</a>
			{/snippet}
		</AppHeader>
	{/snippet}
</Story>

<Story
	name="Anomaly banner"
	play={async ({ canvasElement }) => {
		const banner = within(canvasElement).getByRole('complementary', { name: 'Anomaly' });
		await expect(within(banner).getByRole('link', { name: /Open narrative/ })).toHaveAttribute(
			'href',
			'/narratives/N-0412'
		);
	}}
>
	{#snippet template()}
		<AnomalyBanner {anomaly} href="/narratives/N-0412" />
	{/snippet}
</Story>

<Story
	name="Page header with range toggle"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent('Overview');
		const range = canvas.getByRole('navigation', { name: 'Time range' });
		await expect(within(range).getByRole('link', { current: true })).toHaveTextContent('24 h');
	}}
>
	{#snippet template()}
		<div class="p-8">
			<PageHeader title="Overview" eyebrow="Burkina Faso · updated 14:05 GMT">
				{#snippet actions()}
					<SegmentedLinks label="Time range" options={RANGES} current="24h" />
				{/snippet}
			</PageHeader>
		</div>
	{/snippet}
</Story>

<Story name="Pill filters">
	{#snippet template()}
		<div class="p-8">
			<SegmentedLinks
				label="Topic"
				variant="pills"
				current="misinfo"
				options={[
					{ value: 'misinfo', label: 'Misinformation', href: '?topic=misinfo' },
					{ value: 'all', label: 'All topics', href: '?topic=all' }
				]}
			/>
		</div>
	{/snippet}
</Story>

<Story
	name="App header, views not built yet"
	args={{
		items: [
			{ label: 'Overview', href: '/' },
			{ label: 'Narratives', href: '/narratives', disabled: true },
			{ label: 'High-harm', href: '/safety', count: 3, disabled: true }
		],
		currentHref: '/'
	}}
	play={async ({ canvasElement }) => {
		const nav = within(canvasElement).getByRole('navigation', { name: 'Main' });
		await expect(within(nav).getAllByRole('link')).toHaveLength(1);
		await expect(within(nav).getByText(/High-harm/)).toHaveAttribute('aria-disabled', 'true');
	}}
/>
