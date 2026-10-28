import type { ProcessingRun } from '#lib/domain/monitor.js';
import { monitorSnapshot, monitorWindows, processingRuns } from '#lib/fixtures/monitor.js';
import type { Clock } from '#lib/utils/clock.js';
import { ok } from '../../result.js';
import type { MonitorRepo } from '../../services/monitor.js';
import { simulatedFailure, type StubStore } from './store.js';

/** How long a stub run takes before it shows as finished. */
export const STUB_RUN_MS = 2 * 60 * 1000;

/** A started run, finished with the newest fixture run's numbers once enough time has passed. */
function settled(run: ProcessingRun, now: Date): ProcessingRun {
	if (run.result || now.getTime() - run.startedAt.getTime() < STUB_RUN_MS) return run;
	return { ...run, result: structuredClone(processingRuns[2].result) };
}

export function createStubMonitorRepo(store: StubStore, clock: Clock): MonitorRepo {
	return {
		async getSnapshot(scope, { range }) {
			const simulated = simulatedFailure(scope.accountId);
			if (simulated) return simulated;
			const now = clock.now();
			return ok({
				...structuredClone(monitorSnapshot),
				...structuredClone(monitorWindows[range]),
				runs: [
					...store.monitorRuns.map((run) => settled(run, now)),
					...structuredClone(processingRuns)
				],
				updatedAt: now
			});
		},

		async startRun(scope, { executedBy }) {
			const simulated = simulatedFailure(scope.accountId);
			if (simulated) return simulated;
			const run: ProcessingRun = {
				id: `run-stub-${store.monitorRuns.length + 1}`,
				startedAt: clock.now(),
				type: 'manual',
				executedBy,
				result: null
			};
			store.monitorRuns.unshift(run);
			return ok(structuredClone(run));
		}
	};
}
