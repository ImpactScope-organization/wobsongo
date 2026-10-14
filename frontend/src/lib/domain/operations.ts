/**
 * Operations: the expert review queue, prioritisation, knowledge-base gaps
 * and how well delivered debunks work.
 */
import type { EngineVerdict } from './narrative-detail.js';
import { isHighHarm, type HarmScore } from './narrative.js';
import type { ReviewState } from './review.js';

/** Why an engine result was sent to an expert, besides high harm. */
export type CheckReason = 'spiking' | 'low-confidence' | 'partly-true';

interface ExpertCheckBase {
	id: string;
	/** The claim, e.g. a narrative title. */
	title: string;
	engine: EngineVerdict;
	/** Extra context, e.g. "Mooré audio". */
	note?: string;
	review: ReviewState;
}

/**
 * An engine result sent to an expert: flagged for high harm (with its score),
 * for at least one other reason, or both. Reasons are most important first.
 */
export type ExpertCheck = ExpertCheckBase &
	(
		| { harm: HarmScore; reasons: readonly CheckReason[] }
		| { harm?: undefined; reasons: readonly [CheckReason, ...CheckReason[]] }
	);

/** One active narrative (or topic cluster) placed by harm and spread velocity. */
export interface PriorityPoint {
	id: string;
	/** Null for a topic cluster that is not a single narrative. */
	narrativeId: string | null;
	title: string;
	harm: HarmScore;
	/** Relative spread velocity, 0 (flat) to 1 (fastest in the window). */
	velocity: number;
	volume7d: number;
}

/** Harm at or above this is the upper half of the harm × spread matrix. */
export const PRIORITY_HARM_SPLIT = 3;
/** Velocity at or above this is the right half of the matrix. */
export const PRIORITY_VELOCITY_SPLIT = 0.5;

/** Where a point falls in the harm × spread matrix, and so what to do about it. */
export type PriorityQuadrant = 'critical' | 'urgent' | 'monitor' | 'campaign';

export function priorityQuadrant(
	point: Pick<PriorityPoint, 'harm' | 'velocity'>
): PriorityQuadrant {
	const high = point.harm >= PRIORITY_HARM_SPLIT;
	const fast = point.velocity >= PRIORITY_VELOCITY_SPLIT;
	return high ? (fast ? 'urgent' : 'critical') : fast ? 'campaign' : 'monitor';
}

/** How alarming a point is: high harm, upper half of the matrix, or neither. */
export type PrioritySeverity = 'high' | 'elevated' | 'low';

export function prioritySeverity(point: Pick<PriorityPoint, 'harm'>): PrioritySeverity {
	if (isHighHarm(point.harm)) return 'high';
	return point.harm >= PRIORITY_HARM_SPLIT ? 'elevated' : 'low';
}

/** A cluster of claims the engine could not match to the knowledge base. */
export interface KnowledgeGap {
	id: string;
	label: string;
	/** A representative, anonymized message. */
	example: string;
	messageCount: number;
	languages: readonly string[];
	firstAskedAt: Date;
	review: ReviewState;
}

/** WhatsApp reactions to one delivered debunk. */
export interface DebunkOutcome {
	id: string;
	title: string;
	/** Formats and languages it was delivered in, e.g. "Audio · Mooré, Dioula". */
	formats: string;
	sent: number;
	/** "Was this helpful?" → Yes, in percent. */
	helpfulPct: number;
	/** Asked again on the same topic within 24 h, in percent. */
	followUpPct: number;
}

/** A follow-up rate above this suggests the debunk did not convince. */
export const HIGH_FOLLOW_UP_PCT = 25;

export function isHighFollowUp(outcome: DebunkOutcome): boolean {
	return outcome.followUpPct > HIGH_FOLLOW_UP_PCT;
}
