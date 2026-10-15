/**
 * What services and repositories return. Expected failures (not found, wrong
 * password, validation) are values, not exceptions, so callers must handle
 * them; exceptions are left for bugs and outages.
 */
export interface Failure {
	ok: false;
	/** HTTP-style status, e.g. 400, 404, 409, 501. */
	status: number;
	/** Safe to show to the person. */
	message: string;
}

export type Result<T> = { ok: true; data: T } | Failure;

export function ok<T>(data: T): Result<T> {
	return { ok: true, data };
}

export function failure(status: number, message: string): Failure {
	return { ok: false, status, message };
}

/** For backend calls that aren't wired up yet. */
export function notImplemented(feature: string): Failure {
	return failure(501, `${feature} is not available yet.`);
}
