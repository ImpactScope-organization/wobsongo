/**
 * Sample data for the settings view, taken from the Infodemic Dashboard
 * design. Read-only; shared by stories, tests and stubs.
 */
import type {
	AccountProfile,
	BotConnection,
	BotContact,
	BotLinkCode,
	BotPlatform
} from '#lib/domain/account.js';
import { OVERVIEW_NOW } from './overview.js';

export const SETTINGS_NOW = OVERVIEW_NOW;

export const profile: AccountProfile = {
	name: 'Dr. Aminata Ouédraogo',
	email: 'a.ouedraogo@wobsongo.org',
	role: 'admin'
};

export const passwordChangedAt = new Date('2026-07-08T09:30:00Z');

export const botContacts: Record<BotPlatform, BotContact> = {
	whatsapp: {
		bot: 'whatsapp',
		address: '+226 54 00 21 90',
		chatUrl: (text) => `https://wa.me/22654002190?text=${encodeURIComponent(text)}`
	},
	telegram: {
		bot: 'telegram',
		address: '@WobSongoBot',
		chatUrl: (text) => `https://t.me/WobSongoBot?text=${encodeURIComponent(text)}`
	}
};

export const botConnections: BotConnection[] = [
	{ bot: 'whatsapp', status: 'not-connected' },
	{
		bot: 'telegram',
		status: 'connected',
		handle: '@aminata_o',
		since: new Date('2026-09-02T10:12:00Z')
	}
];

/** Issued 48 s before SETTINGS_NOW, so 9:12 is left. */
export const linkCode: BotLinkCode = {
	bot: 'whatsapp',
	code: '482913',
	expiresAt: new Date(SETTINGS_NOW.getTime() + 9 * 60_000 + 12_000)
};
