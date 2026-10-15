import { failure, type Failure } from '../result.js';

export const SERVER_ERROR_MESSAGE = 'Something went wrong on our side. Try again in a moment.';

/**
 * The message to show for a repository failure: `known` words statuses the
 * operation expects (e.g. 409 on signup); server errors get a generic line so
 * backend details never reach the page. 501 keeps its "not available yet".
 */
export function describeFailure(
	result: Failure,
	known: Partial<Record<number, string>> = {}
): Failure {
	const message =
		known[result.status] ??
		(result.status >= 500 && result.status !== 501 ? SERVER_ERROR_MESSAGE : result.message);
	return failure(result.status, message);
}
