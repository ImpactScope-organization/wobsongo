import { m } from '#lib/paraglide/messages.js';
import { notImplemented } from '../../result.js';
import type { OverviewRepo } from '../../services/overview.js';
import type { ApiOptions } from './index.js';

export function createApiOverviewRepo(_options: ApiOptions): OverviewRepo {
	return {
		// TODO(openapi): GET /api/v1/overview?range=24h|7d|30d → snapshot. Map the
		// backend's verdict codes onto `Verdict` (see domain/narrative.ts) and suppress small
		// counts server-side (domain/count.ts).
		async getSnapshot() {
			return notImplemented(m.nav_overview());
		}
	};
}
