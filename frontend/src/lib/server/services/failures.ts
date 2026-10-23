import { m } from '#lib/paraglide/messages.js';
import { failure, type Failure } from '../result.js';

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
		(result.status >= 500 && result.status !== 501 ? m.error_server() : result.message);
	return failure(result.status, message);
}
