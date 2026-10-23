<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import logo from '#lib/assets/ws-logo-white.svg';
	import { passwordChangedAt, profile, SETTINGS_NOW } from '#lib/fixtures/settings.js';
	import { navItems } from '#lib/navigation.js';
	import AppHeader from '../common/app-header.svelte';
	import BotComingSoon from './bot-coming-soon.svelte';
	import SettingsView from './settings-view.svelte';

	// The settings page as routes/(app)/settings/+page.svelte renders it, inside the app shell.
	const { Story } = defineMeta({
		title: 'Settings/Overview',
		parameters: { layout: 'fullscreen' }
	});

	const viewProps = { profile, passwordChangedAt, now: SETTINGS_NOW };
</script>

<Story
	name="Settings page"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent('Settings');
		await expect(canvas.getAllByRole('heading', { level: 2 })).toHaveLength(3);
		await expect(canvas.getByRole('link', { name: /Settings for/ })).toHaveAttribute(
			'aria-current',
			'page'
		);
		await expect(canvas.getByRole('button', { name: /Connect WhatsApp/ })).toBeDisabled();
		await expect(canvas.getByRole('button', { name: 'Log out' })).toBeEnabled();
	}}
>
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<AppHeader
				logoSrc={logo}
				currentHref="/settings"
				account={{ name: profile.name, href: '/settings' }}
				items={navItems()}
			/>
			<SettingsView {...viewProps} />
		</div>
	{/snippet}
</Story>

<Story name="After a failed profile save">
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<SettingsView
				{...viewProps}
				profileResult={{
					values: { name: 'Aminata O.', email: 'k.sawadogo@wobsongo.org' },
					error: 'Another account already uses this email.'
				}}
			/>
		</div>
	{/snippet}
</Story>

<Story name="After a password change">
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<SettingsView
				{...viewProps}
				passwordChangedAt={SETTINGS_NOW}
				passwordResult={{ updated: true }}
			/>
		</div>
	{/snippet}
</Story>

<Story name="WhatsApp, coming soon">
	{#snippet template()}
		<div class="max-w-[55rem] p-8">
			<BotComingSoon class="rounded-lg border bg-card" />
		</div>
	{/snippet}
</Story>
