/**
 * Narratives are clusters of similar claims seen across platforms and bot
 * submissions. These are the dashboard's own types: the service layer maps the
 * backend's verdict codes onto them, so components don't change when the
 * backend taxonomy does.
 */

/** Fact-check verdict shown to analysts. */
export type Verdict = 'false' | 'misleading' | 'unverified';

export const VERDICTS: readonly Verdict[] = ['false', 'misleading', 'unverified'];

/** Outcome of rating a claim. Narratives are never accurate; rated claims can be. */
export type RatingOutcome = Verdict | 'accurate';

/** Where a narrative sits in the review workflow. */
export type ReviewStatus = 'needs-check' | 'auto-checked' | 'expert-verified' | 'not-in-kb';

export const REVIEW_STATUSES: readonly ReviewStatus[] = [
	'needs-check',
	'auto-checked',
	'expert-verified',
	'not-in-kb'
];

/** Expert-validated harm score, 1 (low) to 5 (dangerous). */
export type HarmScore = 1 | 2 | 3 | 4 | 5;

export const MAX_HARM: HarmScore = 5;

/** Harm at or above this level is treated as high-harm and debunked first. */
export const HIGH_HARM_THRESHOLD: HarmScore = 4;

/** Growth above this percentage, versus the previous window, counts as spiking. */
export const SPIKING_GROWTH_PCT = 50;

export interface NarrativeSummary {
	id: string;
	title: string;
	verdict: Verdict;
	status: ReviewStatus;
	/** SRH topic, e.g. "Family planning". */
	domain: string;
	harm: HarmScore;
	/** Messages in the selected window. */
	volume: number;
	/** Growth versus the previous window, in percent. */
	growthPct: number;
	/** Volume per time bucket over the window, oldest first. */
	trend: number[];
}

export function isHighHarm(harm: HarmScore): boolean {
	return harm >= HIGH_HARM_THRESHOLD;
}

export function isSpiking(growthPct: number): boolean {
	return growthPct > SPIKING_GROWTH_PCT;
}

/** An unusual jump in volume for one narrative. */
export interface Anomaly {
	narrativeId: string;
	narrativeTitle: string;
	/** Standard deviations above the usual level for this hour. */
	sigma: number;
	growthPct: number;
	/** Window the growth is measured over, in hours. */
	windowHours: number;
}

/** Counts behind the high-harm summary card. */
export interface HighHarmSummary {
	/** Dangerous-practice claims awaiting expert review. */
	awaitingReview: number;
	/** Claims fast-tracked to the fact-check queue. */
	fastTracked: number;
}
