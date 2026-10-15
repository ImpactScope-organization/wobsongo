import type { TimeRange } from '#lib/domain/time-range.js';
import { topNarratives, type PulseSnapshot, type PulseTopic } from '#lib/domain/pulse.js';
import { ok, type Result } from '../result.js';
import type { Scope } from '../scope.js';
import { describeFailure } from './failures.js';

/** Where Pulse data comes from: the backend, or a stub. */
export interface PulseRepo {
	/** The window's snapshot, with every narrative active in it. */
	getSnapshot(scope: Scope, query: { range: TimeRange }): Promise<Result<PulseSnapshot>>;
}

export interface PulseService {
	/** The snapshot, with narratives filtered to `topic` and ranked by volume. */
	getPulse(
		scope: Scope,
		query: { range: TimeRange; topic: PulseTopic }
	): Promise<Result<PulseSnapshot>>;
}

/** How many narratives the Pulse list shows. */
export const PULSE_NARRATIVE_LIMIT = 8;

export function createPulseService(repo: PulseRepo): PulseService {
	return {
		async getPulse(scope, { range, topic }) {
			const result = await repo.getSnapshot(scope, { range });
			if (!result.ok) return describeFailure(result, { 404: 'No Pulse data for this account.' });

			const snapshot = result.data;
			return ok({
				...snapshot,
				narratives: topNarratives(snapshot.narratives, topic).slice(0, PULSE_NARRATIVE_LIMIT)
			});
		}
	};
}
