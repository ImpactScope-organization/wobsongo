/**
 * Checks whether an email address is authorized to view unredacted creator data.
 * Whitespace is trimmed and email matching is case-insensitive.
 *
 * @param email The user's email address
 * @param whitelist Comma-separated list of whitelisted emails, or undefined/empty
 */
export function isWhitelistedForFullData(
	email: string | undefined | null,
	whitelist?: string | null
): boolean {
	if (!email) return false;
	const allowed = (whitelist ?? '')
		.split(',')
		.map((e) => e.trim().toLowerCase())
		.filter(Boolean);
	return allowed.includes(email.trim().toLowerCase());
}
