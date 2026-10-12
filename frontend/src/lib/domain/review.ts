/** Decisions an expert can take on an engine result or a knowledge gap. */
export type ReviewDecision = 'approve' | 'override' | 'dismiss' | 'add-fact-check';

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
