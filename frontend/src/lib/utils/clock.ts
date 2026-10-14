/**
 * Wall-clock abstraction. Code that needs "now" takes a Clock instead of
 * calling `new Date()`, so tests, stories and stubs can pin time.
 */
export interface Clock {
	now(): Date;
}

export const systemClock: Clock = {
	now: () => new Date()
};

/** A clock frozen at `at` (a Date or an ISO string). */
export function fixedClock(at: Date | string): Clock {
	const time = new Date(at).getTime();
	return { now: () => new Date(time) };
}
