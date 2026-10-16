/**
 * Everything the narrative detail page shows about one narrative, beyond the
 * summary used in lists.
 */
import type { Count } from './count.js';
import type { NarrativeSummary, Verdict } from './narrative.js';
import type { Platform } from './platform.js';

export interface NarrativeDetail extends NarrativeSummary {
	/** One-sentence plain-language description of the claim. */
	summary: string;
	/** Messages over the last 7 days. */
	volume7d: number;
	/** Volume in the last 6 h divided by the 6 h before, e.g. 4.1. */
	growthFactor6h: number;
	/** Whether a clinical expert has validated the harm score. */
	harmValidated: boolean;
	/** Triage priority, 0 to 100. */
	priority: number;
}

/** Fact-check engine result for a narrative, open to expert review. */
export interface EngineVerdict {
	verdict: Verdict;
	/** 0 to 1. */
	confidence: number;
	/** Why the engine decided this, e.g. which knowledge-base article matched. */
	explanation: string;
}

/** Why an analyst hides a narrative from Overview and alerts. */
export type DismissReason = 'not-srh' | 'not-a-claim' | 'duplicate' | 'satire';

export const DISMISS_REASONS: readonly DismissReason[] = [
	'not-srh',
	'not-a-claim',
	'duplicate',
	'satire'
];

/** Mentions per time bucket on one platform. */
export interface PlatformSeries {
	platform: Platform;
	points: { at: Date; count: number }[];
}

/** How a narrative moved across platforms. */
export interface NarrativeSpread {
	series: PlatformSeries[];
	/** First sighting anywhere. */
	firstSeen: { platform: Platform; at: Date };
	/** First WhatsApp bot submission, if any yet. */
	firstOnWhatsApp: Date | null;
}

/** Hours between first public sighting and first WhatsApp submission, or null. */
export function whatsAppLagHours(spread: NarrativeSpread): number | null {
	if (!spread.firstOnWhatsApp) return null;
	const ms = spread.firstOnWhatsApp.getTime() - spread.firstSeen.at.getTime();
	return Math.max(0, Math.round(ms / (60 * 60 * 1000)));
}

export type MessageFormat = 'audio' | 'video' | 'text' | 'image';

export const MESSAGE_FORMATS: readonly MessageFormat[] = ['audio', 'video', 'text', 'image'];

export interface LanguageCount {
	language: string;
	count: number;
}

export interface FormatCount {
	format: MessageFormat;
	count: number;
}

export interface RegionCount {
	region: string;
	/** Suppressed below the minimum cell size. */
	count: Count;
	/** Most common narrative in the region; omitted when the count is suppressed. */
	topTopic?: string;
}

/** An anonymised message excerpt. */
export interface SampleMessage {
	id: string;
	platform: Platform;
	/** Human description of the format, e.g. "Voice note 0:42". */
	formatLabel: string;
	language: string;
	text: string;
	/** Provenance, e.g. "Auto-transcribed, translated from Mooré". */
	note: string;
}

/** A debunk format suggested for this narrative's audience. */
export interface DebunkFormat {
	id: string;
	format: MessageFormat;
	language: string;
	/**
	 * Share of this narrative's messages in that language, in percent: the audience
	 * a debunk in this format and language would reach.
	 */
	reachPct: number;
	available: boolean;
}
