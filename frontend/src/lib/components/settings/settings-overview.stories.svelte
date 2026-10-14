<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import logo from '#lib/assets/wobsongo-mark.jpg';
	import {
		botConnections,
		botContacts,
		passwordChangedAt,
		profile,
		SETTINGS_NOW
	} from '#lib/fixtures/settings.js';
	import AppHeader from '../common/app-header.svelte';
	import PageHeader from '../common/page-header.svelte';
	import SectionCard from '../common/section-card.svelte';
	import BotConnectionList from './bot-connection-list.svelte';
	import PasswordForm from './password-form.svelte';
	import ProfileForm from './profile-form.svelte';

	// Composition only: how the settings page will assemble these components,
	// for comparing against the design. The real page lives in routes/.
	const { Story } = defineMeta({
		title: 'Settings/Overview',
		parameters: { layout: 'fullscreen' }
	});
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
	}}
>
	{#snippet template()}
		<div class="min-h-screen bg-background">
			<AppHeader
				logoSrc={logo}
				currentHref="/settings"
				account={{ name: profile.name, href: '/settings' }}
				items={[
					{ label: 'Pulse', href: '/pulse' },
					{ label: 'Narratives', href: '/narratives' },
					{ label: 'Spread', href: '/spread' },
					{ label: 'Operations', href: '/ops' },
					{ label: 'High-harm', href: '/safety', count: 3 }
				]}
			/>

			<main class="mx-auto flex max-w-[85rem] flex-col gap-5 p-[clamp(1rem,3vw,2rem)]">
				<PageHeader title="Settings" eyebrow="Your account" class="mb-0.5" />

				<div class="flex max-w-[55rem] flex-col gap-5">
					<SectionCard title="Account" description="Your profile as seen by other reviewers.">
						<ProfileForm {profile} action="?/profile" />
					</SectionCard>

					<SectionCard title="Password">
						<PasswordForm
							action="?/password"
							resetAction="?/resetPassword"
							changedAt={passwordChangedAt}
							now={SETTINGS_NOW}
						/>
					</SectionCard>

					<SectionCard
						title="Bot connections"
						description="Link your own messaging account to the bot to review flagged claims and approve replies from your phone."
						flush
					>
						<BotConnectionList
							connections={botConnections}
							contacts={botContacts}
							connectHrefFor={(bot) => `/settings/bots/${bot}`}
							disconnectHrefFor={(bot) => `/settings/bots/${bot}/disconnect`}
						/>
					</SectionCard>
				</div>
			</main>
		</div>
	{/snippet}
</Story>
