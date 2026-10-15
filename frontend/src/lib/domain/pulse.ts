/**
 * The Pulse view: the headline numbers for one reporting window, plus the
 * narratives, sources and alerts behind them.
 */
import type {
	Anomaly,
	HighHarmSummary,
	NarrativeSummary,
	RatingOutcome,
	Verdict
} from './narrative.js';
import type { PlatformCount } from './platform.js';

/** Which narratives the Pulse list shows. */
export type PulseTopic = 'misinfo' | 'all';

export const PULSE_TOPICS: readonly PulseTopic[] = ['misinfo', 'all'];

export const DEFAULT_PULSE_TOPIC: PulseTopic = 'misinfo';

export function parsePulseTopic(value: string | null | undefined): PulseTopic {
	return PULSE_TOPICS.find((topic) => topic === value) ?? DEFAULT_PULSE_TOPIC;
}

/** Verdicts counted as misinformation; "all" adds unverified claims. */
export const MISINFO_VERDICTS: readonly Verdict[] = ['false', 'misleading'];

export function matchesTopic(verdict: Verdict, topic: PulseTopic): boolean {
	return topic === 'all' || MISINFO_VERDICTS.includes(verdict);
}

/** A value in the selected window next to the same value in the window before it. */
export interface Comparison {
	current: number;
	previous: number;
}

export interface PulseSnapshot {
	/** Country or area the tenant monitors, e.g. "Burkina Faso". */
	region: string;
	updatedAt: Date;
	/** Messages received, with the volume per time bucket, oldest first. */
	messages: Comparison & { trend: number[] };
	/** Narratives active in the window, and how many first appeared in it. */
	activeNarratives: { count: number; new: number };
	/** Narratives rated in the window, by outcome. */
	rated: Record<RatingOutcome, number>;
	medianDebunkMinutes: Comparison;
	/** The window's narratives, in no particular order. */
	narratives: NarrativeSummary[];
	/** The most unusual jump in volume, if any narrative has one. */
	anomaly: Anomaly | null;
	platformCounts: PlatformCount[];
	highHarm: HighHarmSummary;
}

/** Narratives for a topic, highest volume first. */
export function topNarratives(
	narratives: readonly NarrativeSummary[],
	topic: PulseTopic
): NarrativeSummary[] {
	return narratives
		.filter((narrative) => matchesTopic(narrative.verdict, topic))
		.toSorted((a, b) => b.volume - a.volume);
}

/** Change from the previous window in percent, or null when there was nothing before. */
export function growthPct({ current, previous }: Comparison): number | null {
	if (previous === 0) return null;
	return ((current - previous) / previous) * 100;
}

/** Narratives rated false or misleading, out of all rated. */
export function misinfoShare(rated: Record<RatingOutcome, number>): {
	misinfo: number;
	total: number;
} {
	const total = Object.values(rated).reduce((sum, count) => sum + count, 0);
	const misinfo = MISINFO_VERDICTS.reduce((sum, verdict) => sum + rated[verdict], 0);
	return { misinfo, total };
}

/** Direction of a series, comparing its last value with its first. */
export function trendDirection(values: readonly number[]): 'rising' | 'falling' | 'steady' {
	if (values.length < 2) return 'steady';
	const first = values[0];
	const last = values[values.length - 1];
	if (last > first) return 'rising';
	if (last < first) return 'falling';
	return 'steady';
}
