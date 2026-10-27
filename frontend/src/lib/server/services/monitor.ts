import { isRunning, type MonitorSnapshot, type ProcessingRun } from '#lib/domain/monitor.js';
import type { TimeRange } from '#lib/domain/time-range.js';
import { m } from '#lib/paraglide/messages.js';
import { failure, ok, type Result } from '../result.js';
import type { Scope } from '../scope.js';
import { describeFailure } from './failures.js';

/** Where Monitor data comes from: the backend, or a stub. */
export interface MonitorRepo {
	/** The window's figures, with the most recent runs newest first. */
	getSnapshot(scope: Scope, query: { range: TimeRange }): Promise<Result<MonitorSnapshot>>;
	/** Starts a manual run on behalf of `executedBy`. */
	startRun(scope: Scope, run: { executedBy: string }): Promise<Result<ProcessingRun>>;
}

export interface MonitorService {
	/** The snapshot, with at most `MONITOR_RUN_LIMIT` runs, newest first. */
	getMonitor(scope: Scope, query: { range: TimeRange }): Promise<Result<MonitorSnapshot>>;
	/** Starts a manual run, unless one is already going. */
	startRun(scope: Scope, run: { executedBy: string }): Promise<Result<ProcessingRun>>;
}

/** How many runs the Monitor table lists. */
export const MONITOR_RUN_LIMIT = 12;

export function createMonitorService(repo: MonitorRepo): MonitorService {
	return {
		async getMonitor(scope, { range }) {
			const result = await repo.getSnapshot(scope, { range });
			if (!result.ok) return describeFailure(result, { 404: m.monitor_not_found() });

			const snapshot = result.data;
			return ok({
				...snapshot,
				runs: snapshot.runs
					.toSorted((a, b) => b.startedAt.getTime() - a.startedAt.getTime())
					.slice(0, MONITOR_RUN_LIMIT)
			});
		},

		async startRun(scope, { executedBy }) {
			const current = await repo.getSnapshot(scope, { range: '24h' });
			if (!current.ok) return describeFailure(current, { 404: m.monitor_not_found() });
			if (current.data.runs.some(isRunning)) return failure(409, m.monitor_run_already_running());

			const result = await repo.startRun(scope, { executedBy });
			if (!result.ok) return describeFailure(result);
			return result;
		}
	};
}
