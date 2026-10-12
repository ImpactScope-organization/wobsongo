/**
 * Display formatting. Locale and time zone are fixed so server and browser
 * render identical strings (no hydration mismatch) until i18n lands.
 * Burkina Faso is on GMT all year, so UTC is the default zone.
 */
const LOCALE = 'en-US';
export const DEFAULT_TIME_ZONE = 'UTC';

const numberFormat = new Intl.NumberFormat(LOCALE);

export function formatNumber(value: number): string {
	return numberFormat.format(value);
}

/** `12` → "12%"; with `signed`, `12` → "+12%" and `-3` → "−3%". */
export function formatPercent(value: number, { signed = false } = {}): string {
	// `|| 0` turns -0 into 0.
	const rounded = Math.round(value) || 0;
	if (!signed || rounded === 0) return `${rounded}%`;
	return rounded > 0 ? `+${rounded}%` : `−${Math.abs(rounded)}%`;
}

/**
 * Whole-number percentages of `values` that add up to exactly 100, using the
 * largest-remainder method. All zeros (or an empty list) give all zeros.
 */
export function percentShares(values: readonly number[]): number[] {
	const total = values.reduce((sum, value) => sum + value, 0);
	if (total <= 0) return values.map(() => 0);

	const exact = values.map((value) => (value / total) * 100);
	const shares = exact.map(Math.floor);
	let remaining = 100 - shares.reduce((sum, share) => sum + share, 0);

	const byRemainder = exact
		.map((value, index) => ({ index, remainder: value - Math.floor(value) }))
		.sort((a, b) => b.remainder - a.remainder);
	for (const { index } of byRemainder) {
		if (remaining <= 0) break;
		shares[index] += 1;
		remaining -= 1;
	}
	return shares;
}

/** 24-hour clock time, e.g. "14:05". */
export function formatClockTime(at: Date, timeZone = DEFAULT_TIME_ZONE): string {
	return new Intl.DateTimeFormat(LOCALE, {
		hour: '2-digit',
		minute: '2-digit',
		hourCycle: 'h23',
		timeZone
	}).format(at);
}

function dayKey(at: Date, timeZone: string): string {
	return new Intl.DateTimeFormat('en-CA', { dateStyle: 'short', timeZone }).format(at);
}

/**
 * Log timestamp relative to `now`: "Today 14:02", "Yesterday 09:14",
 * otherwise "Oct 9, 14:02".
 */
export function formatLogTime(at: Date, now: Date, timeZone = DEFAULT_TIME_ZONE): string {
	const time = formatClockTime(at, timeZone);
	const day = dayKey(at, timeZone);
	if (day === dayKey(now, timeZone)) return `Today ${time}`;

	const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
	if (day === dayKey(yesterday, timeZone)) return `Yesterday ${time}`;

	const date = new Intl.DateTimeFormat(LOCALE, { month: 'short', day: 'numeric', timeZone }).format(
		at
	);
	return `${date}, ${time}`;
}
