/**
 * Cross-narrative spread: where claims come from, in what format and
 * language, how fast they reach WhatsApp, and why people message the bot.
 */
import type { MessageFormat } from './narrative-detail.js';
import type { Platform } from './platform.js';

/** Counts for one week (starting Monday, UTC), by category. */
export interface WeeklyCounts<K extends string> {
	weekStart: Date;
	counts: Record<K, number>;
}

/** Time from a narrative's first public post to its first WhatsApp question. */
export interface NarrativeLag {
	narrativeId: string;
	title: string;
	/** Public platform where the narrative was first seen. */
	origin: Platform;
	lagHours: number;
}

/** How one language arrives, by format. */
export interface LanguageFormatMix {
	language: string;
	counts: Record<MessageFormat, number>;
}

/** Why someone messaged the bot. */
export type MessageIntent = 'verification' | 'general' | 'out-of-scope';

export const MESSAGE_INTENTS: readonly MessageIntent[] = [
	'verification',
	'general',
	'out-of-scope'
];

export interface DailyIntents {
	day: Date;
	counts: Record<MessageIntent, number>;
}
