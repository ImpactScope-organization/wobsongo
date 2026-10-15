// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
import type { Services, Session } from '#lib/server/services/index.js';
import type { Clock } from '#lib/utils/clock.js';

declare global {
	namespace App {
		// interface Error {}
		interface Locals {
			clock: Clock;
			services: Services;
			/** Who is signed in; never null inside the `(app)` group. */
			session: Session | null;
		}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
