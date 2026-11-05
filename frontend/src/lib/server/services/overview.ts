import type { TimeRange } from '#lib/domain/time-range.js';
import {
	topNarratives,
	type EngagementWindow,
	type OverviewSnapshot,
	type OverviewTopic
} from '#lib/domain/overview.js';
import { m } from '#lib/paraglide/messages.js';
import { ok, type Result } from '../result.js';
import type { Scope } from '../scope.js';
import { describeFailure } from './failures.js';

/** Where Overview data comes from: the backend, or a stub. */
export interface OverviewRepo {
	/** The window's snapshot, with every narrative active in it. */
	getSnapshot(
		scope: Scope,
		query: { range: TimeRange; engagement: EngagementWindow }
	): Promise<Result<OverviewSnapshot>>;
}

export interface OverviewService {
	/** The snapshot, with narratives filtered to `topic` and ranked by volume. */
	getOverview(
		scope: Scope,
		query: { range: TimeRange; engagement: EngagementWindow; topic: OverviewTopic }
	): Promise<Result<OverviewSnapshot>>;
}

/** How many narratives the Overview list shows. */
export const OVERVIEW_NARRATIVE_LIMIT = 8;

export function createOverviewService(repo: OverviewRepo): OverviewService {
	return {
		async getOverview(scope, { range, topic, engagement }) {
			const result = await repo.getSnapshot(scope, { range, engagement });
			if (!result.ok) return describeFailure(result, { 404: m.overview_not_found() });

			const snapshot = result.data;
			return ok({
				...snapshot,
				narratives: topNarratives(snapshot.narratives, topic).slice(0, OVERVIEW_NARRATIVE_LIMIT)
			});
		}
	};
}
