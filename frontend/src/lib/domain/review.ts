/** Decisions an expert can take on an engine result or a knowledge gap. */
export type ReviewDecision = 'approve' | 'override' | 'dismiss' | 'add-fact-check';

/**
 * Values a review form posts as `decision`. `override` is not among them: it needs
 * a new verdict and reply, so it links to an editor page instead of posting.
 */
export type ReviewFormDecision = Exclude<ReviewDecision, 'override'> | 'undo';

export const REVIEW_FORM_DECISIONS = [
	'approve',
	'dismiss',
	'add-fact-check',
	'undo'
] as const satisfies readonly ReviewFormDecision[];

/** Parses a posted `decision` field; null for anything a review form can't send. */
export function parseReviewFormDecision(value: unknown): ReviewFormDecision | null {
	return REVIEW_FORM_DECISIONS.find((decision) => decision === value) ?? null;
}

/** State of one review item: still pending, or decided (and undoable). */
export type ReviewState = { kind: 'pending' } | { kind: 'decided'; decision: ReviewDecision };

/** One line in an audit or activity log. */
export interface ActivityEntry {
	id: string;
	at: Date;
	/** Person's display name, or null for system events. */
	actor: string | null;
	text: string;
}
