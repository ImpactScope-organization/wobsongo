import { m } from '#lib/paraglide/messages.js';
import { notImplemented } from '../../result.js';
import type { MonitorRepo } from '../../services/monitor.js';
import type { ApiOptions } from './index.js';

export function createApiMonitorRepo(_options: ApiOptions): MonitorRepo {
	return {
		// TODO(openapi): GET /api/v1/monitor?range=24h|7d|30d → runs, window totals and
		// volume buckets. Map the pipeline's post statuses onto `ProcessingStatus`
		// (see domain/monitor.ts).
		async getSnapshot() {
			return notImplemented(m.nav_monitor());
		},
		// TODO(openapi): POST /api/v1/monitor/runs → the started run.
		async startRun() {
			return notImplemented(m.nav_monitor());
		}
	};
}
