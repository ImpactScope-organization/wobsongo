/**
 * Dangerous-practice alerts: claims promoting toxic substances or unsafe
 * procedures, matched against the clinical safety lexicon.
 */
import type { EngineVerdict } from './narrative-detail.js';
import type { Platform } from './platform.js';
import type { ReviewState } from './review.js';

/** Clinical risk of the practice a claim promotes. */
export type HarmRisk = 'critical' | 'high';

export interface HarmAlert {
	id: string;
	/** Anonymized message excerpt (translated where needed). */
	quote: string;
	/** Lexicon match, e.g. "Potash (kanwa) + lemon · ingestion". */
	substance: string;
	risk: HarmRisk;
	/** Whether a medical expert has already assessed the risk. */
	riskReviewed: boolean;
	platform: Platform;
	/** Channel and format detail, e.g. "voice note" or "public group". */
	channel: string;
	language: string;
	region: string;
	/** Other messages clustered with this one. */
	similarCount: number;
	detectedAt: Date;
	/** The engine's reply, which a medical expert should check. */
	engine: EngineVerdict;
	review: ReviewState;
}

/** One substance or practice from the safety lexicon, with its mention count. */
export interface SubstanceMention {
	id: string;
	name: string;
	/** How it is used, e.g. "Ingestion" or "Douching". */
	practice: string;
	count: number;
}

/** Mentions at or above this, in the window, mark a substance as frequent. */
export const FREQUENT_MENTION_COUNT = 9;

export function isFrequentMention(mention: SubstanceMention): boolean {
	return mention.count >= FREQUENT_MENTION_COUNT;
}
