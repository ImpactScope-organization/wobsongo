/**
 * The signed-in reviewer's account: profile, password rules and the
 * messaging bots they can link to review claims from their phone.
 */

/**
 * What a person may do in the dashboard, lowest first. Everyone who signs up
 * is a `user`; an administrator promotes them.
 */
export type AccountRole = 'user' | 'admin' | 'superadmin';

export const ACCOUNT_ROLES: readonly AccountRole[] = ['user', 'admin', 'superadmin'];

export interface AccountProfile {
	name: string;
	email: string;
	/** Set by an administrator; read-only for the person. */
	role: AccountRole;
}

/** Passwords shorter than this are rejected. */
export const PASSWORD_MIN_LENGTH = 12;

/** Longest display name accepted. */
export const NAME_MAX_LENGTH = 100;

/** Trimmed and lowercased, so the same address can't sign up twice. */
export function normalizeEmail(email: string): string {
	return email.trim().toLowerCase();
}

// Deliberately loose: the real check is the email the server sends.
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** Why a name and email can't be saved, or null if they can. */
export function profileError({ name, email }: { name: string; email: string }): string | null {
	if (!name.trim()) return 'Enter your name.';
	if (name.trim().length > NAME_MAX_LENGTH)
		return `Use at most ${NAME_MAX_LENGTH} characters for your name.`;
	if (!EMAIL_PATTERN.test(email.trim())) return 'Enter a valid email address.';
	return null;
}

/** Why a new password can't be used, or null if it can. */
export function newPasswordError({
	newPassword,
	confirmPassword
}: {
	newPassword: string;
	confirmPassword: string;
}): string | null {
	if (newPassword.length < PASSWORD_MIN_LENGTH)
		return `Use at least ${PASSWORD_MIN_LENGTH} characters for your password.`;
	if (newPassword !== confirmPassword) return 'The passwords don’t match.';
	return null;
}

/** Why a password change can't go ahead, or null if it can. */
export function passwordChangeError(change: {
	currentPassword: string;
	newPassword: string;
	confirmPassword: string;
}): string | null {
	if (!change.currentPassword) return 'Enter your current password.';
	const error = newPasswordError(change);
	if (error) return error;
	if (change.newPassword === change.currentPassword)
		return 'Choose a password different from your current one.';
	return null;
}

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
