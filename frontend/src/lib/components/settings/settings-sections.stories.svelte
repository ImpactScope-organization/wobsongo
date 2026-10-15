<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, userEvent, within } from 'storybook/test';
	import {
		botConnections,
		botContacts,
		linkCode,
		passwordChangedAt,
		profile,
		SETTINGS_NOW
	} from '#lib/fixtures/settings.js';
	import SectionCard from '../common/section-card.svelte';
	import * as Dialog from '../ui/dialog/index.js';
	import BotConnectPanel from './bot-connect-panel.svelte';
	import BotConnectionList from './bot-connection-list.svelte';
	import BotDisconnectPanel from './bot-disconnect-panel.svelte';
	import PasswordForm from './password-form.svelte';
	import ProfileForm from './profile-form.svelte';

	/** Stops the real submit (it would navigate the story) and records what would be posted. */
	function captureSubmit(form: HTMLFormElement) {
		const posted: { data?: FormData } = {};
		form.addEventListener('submit', (event) => {
			event.preventDefault();
			posted.data = new FormData(form);
		});
		return posted;
	}

	const panelProps = {
		contact: botContacts.whatsapp,
		code: linkCode,
		now: SETTINGS_NOW,
		sentAction: '?/sent',
		regenerateAction: '?/regenerate',
		closeHref: '/settings',
		checkHref: '/settings/bots/whatsapp'
	};

	const { Story } = defineMeta({
		title: 'Settings/Sections',
		component: ProfileForm,
		tags: ['autodocs'],
		parameters: { layout: 'padded' },
		args: { profile, action: '?/profile' }
	});
</script>

<Story
	name="Account"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const posted = captureSubmit(canvasElement.querySelector('form')!);
		const name = canvas.getByLabelText('Full name');
		await userEvent.clear(name);
		await userEvent.type(name, 'Aminata Ouédraogo');
		await userEvent.click(canvas.getByRole('button', { name: 'Save changes' }));
		await expect(posted.data?.get('name')).toBe('Aminata Ouédraogo');
		await expect([...posted.data!.keys()].toSorted()).toEqual(['email', 'name']);
	}}
>
	{#snippet template(args)}
		<SectionCard
			title="Account"
			description="Your profile as seen by other reviewers."
			class="max-w-[55rem]"
		>
			<ProfileForm {...args} />
		</SectionCard>
	{/snippet}
</Story>

<Story name="Account, saved" args={{ saved: true }}>
	{#snippet template(args)}
		<SectionCard title="Account" class="max-w-[55rem]"><ProfileForm {...args} /></SectionCard>
	{/snippet}
</Story>

<Story
	name="Password"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const [passwordForm, resetForm] = canvasElement.querySelectorAll('form');
		const password = captureSubmit(passwordForm);
		const reset = captureSubmit(resetForm);

		await userEvent.click(canvas.getByRole('button', { name: /Forgot your password/ }));
		await expect(reset.data).toBeDefined();
		await expect([...reset.data!.keys()]).toEqual([]);
		await expect(password.data).toBeUndefined();

		await userEvent.type(canvas.getByLabelText('Current password'), 'old password 1');
		await userEvent.type(canvas.getByLabelText('New password'), 'a much longer one');
		await userEvent.type(canvas.getByLabelText('Confirm new password'), 'a much longer one');
		await userEvent.click(canvas.getByRole('button', { name: 'Update password' }));
		await expect(password.data?.get('newPassword')).toBe('a much longer one');
	}}
>
	{#snippet template()}
		<SectionCard title="Password" class="max-w-[55rem]">
			<PasswordForm
				action="?/password"
				resetAction="?/resetPassword"
				changedAt={passwordChangedAt}
				now={SETTINGS_NOW}
			/>
		</SectionCard>
	{/snippet}
</Story>

<Story name="Password, reset link sent">
	{#snippet template()}
		<SectionCard title="Password" class="max-w-[55rem]">
			<PasswordForm
				action="?/password"
				resetAction="?/resetPassword"
				changedAt={passwordChangedAt}
				now={SETTINGS_NOW}
				resetSentTo={profile.email}
			/>
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Bot connections"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('link', { name: 'Connect WhatsApp' })).toHaveAttribute(
			'href',
			'/settings/bots/whatsapp'
		);
		await expect(canvas.getByRole('link', { name: 'Disconnect Telegram' })).toHaveAttribute(
			'href',
			'/settings/bots/telegram/disconnect'
		);
	}}
>
	{#snippet template()}
		<SectionCard
			title="Bot connections"
			description="Link your own messaging account to the bot to review flagged claims and approve replies from your phone."
			flush
			class="max-w-[55rem]"
		>
			<BotConnectionList
				connections={botConnections}
				contacts={botContacts}
				connectHrefFor={(bot) => `/settings/bots/${bot}`}
				disconnectHrefFor={(bot) => `/settings/bots/${bot}/disconnect`}
			/>
		</SectionCard>
	{/snippet}
</Story>

<Story
	name="Connect, code"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('WOBSONGO-AUTH 482-913')).toBeVisible();
		// The countdown runs in the browser.
		const time = canvas.getByText(/^\d:\d\d$/);
		const before = time.textContent;
		await new Promise((resolve) => setTimeout(resolve, 1100));
		await expect(time.textContent).not.toBe(before);
		// Copy appears once the clipboard is known to work.
		await expect(canvas.getByRole('button', { name: 'Copy message' })).toBeVisible();
	}}
>
	{#snippet template()}
		<div class="max-w-[30rem] overflow-hidden rounded-xl bg-card shadow-2xl">
			<BotConnectPanel {...panelProps} phase="code" />
		</div>
	{/snippet}
</Story>

<Story name="Connect, in a dialog">
	{#snippet template()}
		<Dialog.Root open>
			<Dialog.Content
				showCloseButton={false}
				aria-labelledby="connect-title"
				class="gap-0 overflow-hidden rounded-xl bg-card p-0 sm:max-w-[30rem]"
			>
				<BotConnectPanel {...panelProps} phase="code" titleId="connect-title" />
			</Dialog.Content>
		</Dialog.Root>
	{/snippet}
</Story>

<Story name="Connect, expired">
	{#snippet template()}
		<div class="max-w-[30rem] overflow-hidden rounded-xl bg-card shadow-2xl">
			<BotConnectPanel
				{...panelProps}
				phase="code"
				now={new Date(linkCode.expiresAt.getTime() + 1000)}
			/>
		</div>
	{/snippet}
</Story>

<Story name="Connect, waiting">
	{#snippet template()}
		<div class="max-w-[30rem] overflow-hidden rounded-xl bg-card shadow-2xl">
			<BotConnectPanel {...panelProps} phase="waiting" />
		</div>
	{/snippet}
</Story>

<Story name="Connect, connected">
	{#snippet template()}
		<div class="max-w-[30rem] overflow-hidden rounded-xl bg-card shadow-2xl">
			<BotConnectPanel {...panelProps} phase="connected" />
		</div>
	{/snippet}
</Story>

<Story
	name="Disconnect"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const posted = captureSubmit(canvasElement.querySelector('form')!);
		await userEvent.click(canvas.getByRole('button', { name: 'Disconnect' }));
		await expect(posted.data?.get('bot')).toBe('telegram');
	}}
>
	{#snippet template()}
		<div class="max-w-[26.25rem] rounded-xl bg-card shadow-2xl">
			<BotDisconnectPanel bot="telegram" action="?/disconnect" cancelHref="/settings" />
		</div>
	{/snippet}
</Story>
