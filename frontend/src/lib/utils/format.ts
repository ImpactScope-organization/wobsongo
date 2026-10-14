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
/** "n = 902", or "n ≈ 900" for a total rounded to protect suppressed counts. */
export function formatTotal(total: { value: number; approximate: boolean }, label = 'n'): string {
	return `${label} ${total.approximate ? '≈' : '='} ${formatNumber(total.value)}`;
}

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

/** Short weekday and clock time, e.g. "Mon 02:10". */
export function formatDayTime(at: Date, timeZone = DEFAULT_TIME_ZONE): string {
	const day = new Intl.DateTimeFormat(LOCALE, { weekday: 'short', timeZone }).format(at);
	return `${day} ${formatClockTime(at, timeZone)}`;
}

/** Growth multiplier, e.g. 4.1 → "×4.1". */
export function formatFactor(value: number): string {
	return `×${value.toFixed(1)}`;
}

/**
 * Elapsed time before `now`, coarsened: "just now", "38 min ago", "5 h ago",
 * "3 d ago". Future times read as "just now".
 */
export function formatAgo(at: Date, now: Date): string {
	const minutes = Math.floor((now.getTime() - at.getTime()) / 60_000);
	if (minutes < 1) return 'just now';
	if (minutes < 60) return `${minutes} min ago`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${hours} h ago`;
	return `${Math.floor(hours / 24)} d ago`;
}

/**
 * Coarse age for long spans: "today", "yesterday", "5 days ago", "2 weeks ago",
 * "3 months ago", "1 year ago". Calendar months and years are approximated.
 */
export function formatSince(at: Date, now: Date): string {
	const days = Math.floor((now.getTime() - at.getTime()) / 86_400_000);
	const plural = (count: number, unit: string) => `${count} ${unit}${count === 1 ? '' : 's'} ago`;
	if (days < 1) return 'today';
	if (days < 2) return 'yesterday';
	if (days < 14) return plural(days, 'day');
	if (days < 60) return plural(Math.floor(days / 7), 'week');
	if (days < 365) return plural(Math.floor(days / 30), 'month');
	return plural(Math.floor(days / 365), 'year');
}

/** Calendar date, e.g. "2 Sep 2026". */
export function formatDate(at: Date, timeZone = DEFAULT_TIME_ZONE): string {
	// Assembled from parts: en-GB would give "Sept", en-US "Sep 2, 2026".
	const parts = new Intl.DateTimeFormat(LOCALE, {
		day: 'numeric',
		month: 'short',
		year: 'numeric',
		timeZone
	}).formatToParts(at);
	const part = (type: Intl.DateTimeFormatPartTypes) =>
		parts.find((item) => item.type === type)?.value ?? '';
	return `${part('day')} ${part('month')} ${part('year')}`;
}

/** Minutes and seconds left, e.g. "9:05"; never negative. */
export function formatCountdown(ms: number): string {
	const seconds = Math.max(0, Math.ceil(ms / 1000));
	return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;
}

/** Short weekday, e.g. "Mon". */
export function formatWeekday(at: Date, timeZone = DEFAULT_TIME_ZONE): string {
	return new Intl.DateTimeFormat(LOCALE, { weekday: 'short', timeZone }).format(at);
}

/** ISO 8601 week number of a UTC date, e.g. "W34". */
export function formatIsoWeek(at: Date): string {
	const day = new Date(Date.UTC(at.getUTCFullYear(), at.getUTCMonth(), at.getUTCDate()));
	// Move to the Thursday of this week; its year owns the week.
	day.setUTCDate(day.getUTCDate() + 3 - ((day.getUTCDay() + 6) % 7));
	const firstThursday = new Date(Date.UTC(day.getUTCFullYear(), 0, 4));
	firstThursday.setUTCDate(firstThursday.getUTCDate() + 3 - ((firstThursday.getUTCDay() + 6) % 7));
	return `W${1 + Math.round((day.getTime() - firstThursday.getTime()) / (7 * 86_400_000))}`;
}
