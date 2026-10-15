import type { Clock } from '#lib/utils/clock.js';

/** A clock that only moves when told to, for testing expiry. */
export interface ManualClock extends Clock {
	advance(ms: number): void;
}

export function manualClock(start: Date | string): ManualClock {
	let time = new Date(start).getTime();
	return {
		now: () => new Date(time),
		advance: (ms) => {
			time += ms;
		}
	};
}
