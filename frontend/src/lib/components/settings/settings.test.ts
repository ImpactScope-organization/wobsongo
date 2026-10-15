import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { BOT_LINK_CODE_TTL_MS } from '#lib/domain/account.js';
import {
	botConnections,
	botContacts,
	linkCode,
	passwordChangedAt,
	profile,
	SETTINGS_NOW
} from '#lib/fixtures/settings.js';
import { stripComments } from '#lib/testing/snippet.js';
import AppHeader from '../common/app-header.svelte';
import BotComingSoon from './bot-coming-soon.svelte';
import BotConnectPanel from './bot-connect-panel.svelte';
import BotConnectionList from './bot-connection-list.svelte';
import BotDisconnectPanel from './bot-disconnect-panel.svelte';
import PasswordForm from './password-form.svelte';
import ProfileForm from './profile-form.svelte';
import SettingsView from './settings-view.svelte';

const input = (body: string, name: string) =>
	body.match(new RegExp(`<input(?=[^>]*name="${name}")[^>]*>`))?.[0] ?? '';

describe('AppHeader account link', () => {
	const items = [{ label: 'Pulse', href: '/pulse' }];

	it('links the avatar to settings and names it for screen readers', () => {
		const { body } = render(AppHeader, {
			props: { items, currentHref: '/pulse', account: { name: profile.name, href: '/settings' } }
		});
		expect(body).toMatch(/<a href="\/settings"(?![^>]*aria-current)[^>]*>/);
		expect(body).toContain('>AO</span>');
		expect(body).toContain('Settings for Dr. Aminata Ouédraogo');
	});

	it('marks it current on the settings page', () => {
		const { body } = render(AppHeader, {
			props: { items, currentHref: '/settings', account: { name: profile.name, href: '/settings' } }
		});
		expect(body).toMatch(/<a href="\/settings" aria-current="page"/);
	});
});

describe('ProfileForm', () => {
	const renderForm = (props = {}) =>
		stripComments(render(ProfileForm, { props: { profile, action: '?/profile', ...props } }).body);

	it('shows who the person is and their role', () => {
		const body = renderForm();
		expect(body).toContain('Dr. Aminata Ouédraogo');
		expect(body).toContain('a.ouedraogo@wobsongo.org');
		expect(body).toContain('Administrator');
	});

	it('posts name and email, never the role', () => {
		const body = renderForm();
		expect(body).toMatch(/<form[^>]*method="POST"[^>]*action="\?\/profile"/);
		expect(input(body, 'name')).toContain('value="Dr. Aminata Ouédraogo"');
		expect(input(body, 'email')).toContain('value="a.ouedraogo@wobsongo.org"');
		expect(body).not.toMatch(/name="role"/);
	});

	it('refills submitted values and announces errors and success', () => {
		const failed = renderForm({ values: { email: 'not-an-email' }, error: 'Enter a valid email.' });
		expect(input(failed, 'email')).toContain('value="not-an-email"');
		expect(failed).toMatch(/role="alert"[^>]*>\s*Enter a valid email\./);
		expect(renderForm({ saved: true })).toMatch(/role="status"[^>]*>\s*✓ Changes saved/);
	});
});

describe('PasswordForm', () => {
	const renderForm = (props = {}) =>
		stripComments(
			render(PasswordForm, {
				props: {
					action: '?/password',
					resetAction: '?/resetPassword',
					changedAt: passwordChangedAt,
					now: SETTINGS_NOW,
					...props
				}
			}).body
		);

	it('says when the password last changed', () => {
		expect(renderForm()).toMatch(
			/Last changed <time datetime="2026-07-08[^"]*">3 months ago<\/time>/
		);
		expect(renderForm({ changedAt: null })).toContain('Never changed.');
	});

	it('posts the three passwords and enforces the minimum length', () => {
		const body = renderForm();
		expect(input(body, 'currentPassword')).toContain('autocomplete="current-password"');
		expect(input(body, 'newPassword')).toContain('minlength="12"');
		expect(input(body, 'newPassword')).toContain('placeholder="At least 12 characters"');
		expect(input(body, 'confirmPassword')).toContain('minlength="12"');
	});

	it('sends the reset request through its own form, without the passwords', () => {
		const body = renderForm();
		const formId = body.match(/form="(reset-[^"]+)"[^>]*>\s*Forgot your password/)?.[1];
		expect(formId).toBeTruthy();
		expect(body).toMatch(
			new RegExp(`<form id="${formId}" method="POST" action="\\?\\/resetPassword"[^>]*><\\/form>`)
		);
	});

	it('confirms a sent reset link and a changed password', () => {
		expect(renderForm({ resetSentTo: profile.email })).toContain(
			'✓ Reset link sent to a.ouedraogo@wobsongo.org'
		);
		expect(renderForm({ updated: true })).toContain('✓ Password updated');
	});
});

describe('BotConnectionList', () => {
	const renderList = () =>
		stripComments(
			render(BotConnectionList, {
				props: {
					connections: botConnections,
					contacts: botContacts,
					connectHrefFor: (bot: string) => `/settings/bots/${bot}`,
					disconnectHrefFor: (bot: string) => `/settings/bots/${bot}/disconnect`
				}
			}).body
		);

	it('shows how to reach a bot that is not connected, with a connect link', () => {
		const body = renderList();
		expect(body).toContain('Bot: +226 54 00 21 90');
		expect(body).toMatch(
			/href="\/settings\/bots\/whatsapp"[^>]*>\s*Connect<span class="sr-only">(&nbsp;|\u00a0)WhatsApp/
		);
	});

	it('shows the linked account and since when, with a disconnect link', () => {
		const body = renderList();
		expect(body).toMatch(
			/Linked to @aminata_o · since\s*<time datetime="2026-09-02[^"]*">2 Sep 2026/
		);
		expect(body).toContain('href="/settings/bots/telegram/disconnect"');
		expect(body.match(/data-status="connected"/g)).toHaveLength(1);
	});
});

describe('BotConnectPanel', () => {
	const renderPanel = (props = {}) =>
		stripComments(
			render(BotConnectPanel, {
				props: {
					contact: botContacts.whatsapp,
					phase: 'code',
					code: linkCode,
					now: SETTINGS_NOW,
					sentAction: '?/sent',
					regenerateAction: '?/regenerate',
					closeHref: '/settings',
					checkHref: '/settings/bots/whatsapp',
					...props
				}
			}).body
		);

	it('shows the message to send, with its time left and a prefilled chat link', () => {
		const body = renderPanel();
		expect(body).toContain('WOBSONGO-AUTH 482-913');
		expect(body).toMatch(/Expires in <time datetime="PT552S">9:12<\/time>/);
		expect(body).toContain('href="https://wa.me/22654002190?text=WOBSONGO-AUTH%20482-913"');
		expect(body).toContain('rel="noopener noreferrer"');
	});

	it('offers copying only once JavaScript shows the clipboard works', () => {
		expect(renderPanel()).not.toContain('Copy message');
	});

	it('posts "I’ve sent it" and "Generate new code" as plain forms', () => {
		const body = renderPanel();
		expect(body).toMatch(/action="\?\/sent"[^>]*>\s*<button type="submit"[^>]*>I’ve sent it/);
		expect(body).toMatch(
			/action="\?\/regenerate"[^>]*>\s*<button type="submit"[^>]*>\s*Generate new code/
		);
	});

	it('stops offering an expired code', () => {
		const body = renderPanel({
			now: new Date(linkCode.expiresAt.getTime() + 1000)
		});
		expect(body).toContain('data-expired');
		expect(body).toContain('The code has expired.');
		expect(body).not.toContain('I’ve sent it');
		expect(body).not.toContain('href="https://wa.me');
	});

	it('waits for the bot with a way to check again, then confirms', () => {
		const waiting = renderPanel({ phase: 'waiting', code: undefined });
		expect(waiting).toMatch(/role="status"[^>]*>Waiting for the bot…/);
		expect(waiting).toContain('href="/settings/bots/whatsapp"');
		const done = renderPanel({ phase: 'connected', code: undefined });
		expect(done).toMatch(/role="status"[^>]*>Account connected/);
		expect(done).toContain('You can now review claims from WhatsApp.');
	});

	it('titles itself for a page or a dialog', () => {
		expect(renderPanel({ level: 1 })).toMatch(/<h1 id="[^"]+"[^>]*>Connect WhatsApp<\/h1>/);
		expect(renderPanel()).toMatch(/<h2 id="[^"]+"[^>]*>Connect WhatsApp<\/h2>/);
	});

	it('uses a ten-minute code lifetime', () => {
		expect(BOT_LINK_CODE_TTL_MS).toBe(600_000);
	});
});

describe('BotDisconnectPanel', () => {
	it('asks to confirm and posts the bot', () => {
		const body = stripComments(
			render(BotDisconnectPanel, {
				props: { bot: 'telegram', action: '?/disconnect', cancelHref: '/settings' }
			}).body
		);
		expect(body).toContain('Disconnect Telegram?');
		expect(body).toMatch(/<form[^>]*method="POST"[^>]*action="\?\/disconnect"/);
		expect(body).toContain('name="bot" value="telegram"');
		expect(body).toMatch(/href="\/settings"[^>]*>Cancel/);
	});
});

describe('BotComingSoon', () => {
	it('shows WhatsApp as coming soon with an example code and a disabled button', () => {
		const body = stripComments(render(BotComingSoon).body);
		expect(body).toContain('WhatsApp');
		expect(body).toContain('Coming soon');
		expect(body).toContain('WOBSONGO-AUTH 123-456');
		expect(body).toMatch(/<button type="button" disabled=""[^>]*>\s*Connect/);
		expect(body).not.toContain('<a ');
	});
});

describe('SettingsView', () => {
	const props = { profile, passwordChangedAt, now: SETTINGS_NOW };

	it('posts each section to its own action, and logs out with a POST', () => {
		const body = render(SettingsView, { props }).body;
		expect(body).toContain('action="?/profile"');
		expect(body).toContain('action="?/password"');
		expect(body).toContain('action="?/resetPassword"');
		expect(body).toMatch(/<form method="POST" action="\/logout">/);
		expect(body).toContain('Coming soon');
	});

	it('shows each result in its own section only', () => {
		const saved = render(SettingsView, {
			props: { ...props, profileResult: { saved: true } }
		}).body;
		expect(saved).toContain('Changes saved');
		expect(saved).not.toContain('Password updated');

		const updated = render(SettingsView, {
			props: { ...props, passwordResult: { updated: true } }
		}).body;
		expect(updated).toContain('Password updated');
		expect(updated).not.toContain('Changes saved');
	});

	it('refills the profile form after a failed save', () => {
		const body = render(SettingsView, {
			props: {
				...props,
				profileResult: { values: { name: 'Typed name', email: 'typed@x.org' }, error: 'Nope.' }
			}
		}).body;
		expect(input(body, 'name')).toContain('value="Typed name"');
		expect(body).toContain('Nope.');
	});
});
