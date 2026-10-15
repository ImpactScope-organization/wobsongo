/**
 * Stub repositories: fixture data plus an in-memory store, no backend. Used
 * when API_HOST starts with `stub://`, and by tests.
 */
import type { Clock } from '#lib/utils/clock.js';
import type { Repos } from '../../services/index.js';
import { createStubAccountRepo } from './account.js';
import { createStubAuthRepo } from './auth.js';
import { createStubPulseRepo } from './pulse.js';
import { createStubStore, type StubStore } from './store.js';

export function createStubRepos(clock: Clock, store: StubStore = createStubStore()): Repos {
	return {
		auth: createStubAuthRepo(store, clock),
		account: createStubAccountRepo(store, clock),
		pulse: createStubPulseRepo(clock)
	};
}

export {
	createStubStore,
	DEMO_ACCOUNT_ID,
	DEMO_EMAIL,
	DEMO_PASSWORD,
	DEMO_TENANT_ID,
	ERROR_EMAIL_DOMAIN,
	type StubStore
} from './store.js';
