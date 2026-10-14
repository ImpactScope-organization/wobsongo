/**
 * The signed-in reviewer's account: profile, password rules and the
 * messaging bots they can link to review claims from their phone.
 */

/** What a person may do in the dashboard. */
export type AccountRole = 'expert' | 'analyst' | 'admin';

export interface AccountProfile {
	name: string;
	email: string;
	/** Set by an administrator; read-only for the person. */
	organization: string;
	role: AccountRole;
	/** Free text shown under the name, e.g. "Clinical advisor · Reproductive health". */
	title: string;
}

/** Passwords shorter than this are rejected. */
export const PASSWORD_MIN_LENGTH = 12;

/** Up to two initials for an avatar: "Dr. Aminata Ouédraogo" → "AO". */
export function initials(name: string): string {
	const words = name
		.split(/\s+/)
		.filter((word) => word && !/\.$/.test(word))
		.map((word) => [...word][0]?.toUpperCase() ?? '');
	if (words.length === 0) return '?';
	return words.length === 1 ? words[0] : words[0] + words[words.length - 1];
}

/** Messaging bots a reviewer can link their own account to. */
export type BotPlatform = 'whatsapp' | 'telegram';

export const BOT_PLATFORMS: readonly BotPlatform[] = ['whatsapp', 'telegram'];

/** How to reach the bot, e.g. "+226 54 00 21 90" or "@WobSongoBot". */
export interface BotContact {
	bot: BotPlatform;
	address: string;
	/** Chat link that opens the app with `text` prefilled. */
	chatUrl: (text: string) => string;
}

export type BotConnection =
	| {
			bot: BotPlatform;
			status: 'connected';
			/** The linked account, masked by the server, e.g. "+226 70 •• •• 41". */
			handle: string;
			since: Date;
	  }
	| { bot: BotPlatform; status: 'not-connected' };

/** A link code lives this long. */
export const BOT_LINK_CODE_TTL_MS = 10 * 60_000;

/** Prefix that tells the bot a message is a login code, never a question to fact-check. */
export const BOT_LINK_PREFIX = 'WOBSONGO-AUTH';

/** A single-use code, issued by the server, that links a messaging account. */
export interface BotLinkCode {
	bot: BotPlatform;
	/** Six digits. */
	code: string;
	expiresAt: Date;
}

/** The message to send to the bot: "WOBSONGO-AUTH 123-456". */
export function botLinkMessage(code: Pick<BotLinkCode, 'code'>): string {
	return `${BOT_LINK_PREFIX} ${code.code.slice(0, 3)}-${code.code.slice(3)}`;
}

/** Where a link attempt stands: code shown, waiting for the bot, or done. */
export type BotLinkPhase = 'code' | 'waiting' | 'connected';
