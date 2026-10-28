/**
 * The Monitor view: the crawler's processing runs, what happened to the posts
 * each run discovered, and the processing volume over a reporting window.
 */

/** Runs start every 2 h on their own, or when an administrator asks for one. */
export type RunType = 'automatic' | 'manual';

/** Public platforms the crawler discovers posts on. */
export type CrawledPlatform = 'facebook' | 'tiktok' | 'x' | 'instagram' | 'youtube';

export const CRAWLED_PLATFORMS: readonly CrawledPlatform[] = [
	'facebook',
	'tiktok',
	'x',
	'instagram',
	'youtube'
];

/** Pipeline stages a discovered post moves through, in order. */
export type ProcessingStatus =
	| 'discovered'
	| 'screened_out'
	| 'screened_in'
	| 'transcribed'
	| 'out_of_scope'
	| 'analyzed'
	| 'verified'
	| 'clustered'
	| 'failed';

export const PROCESSING_STATUSES: readonly ProcessingStatus[] = [
	'discovered',
	'screened_out',
	'screened_in',
	'transcribed',
	'out_of_scope',
	'analyzed',
	'verified',
	'clustered',
	'failed'
];

/** Statuses a post stays in; the others are steps on the way. */
export const FINAL_STATUSES: readonly ProcessingStatus[] = [
	'screened_out',
	'out_of_scope',
	'clustered',
	'failed'
];

export function isFinalStatus(status: ProcessingStatus): boolean {
	return FINAL_STATUSES.includes(status);
}

/** What a finished run did. */
export interface RunResult {
	/** Posts the crawler found, including ones seen in earlier runs. */
	discovered: number;
	/** Posts found for the first time. */
	newPosts: number;
	durationMinutes: number;
	/** New posts whose processing failed. */
	failed: number;
	/** Estimated processing cost in US dollars. */
	costUsd: number;
	/** New posts by their current status; sums to `newPosts`. */
	statusCounts: Record<ProcessingStatus, number>;
	/** New posts by platform; sums to `newPosts`. */
	platformCounts: Record<CrawledPlatform, number>;
}

export interface ProcessingRun {
	id: string;
	startedAt: Date;
	type: RunType;
	/** Who started a manual run; null for automatic runs. */
	executedBy: string | null;
	/** null while the run is still going. */
	result: RunResult | null;
}

/** Posts processed in one time bucket. */
export interface VolumeBucket {
	start: Date;
	processed: number;
	failed: number;
}

export interface MonitorSnapshot {
	updatedAt: Date;
	/** The most recent runs, newest first. */
	runs: ProcessingRun[];
	/** Runs in the window, which may be more than `runs` lists. */
	runCount: number;
	/** Estimated cost of the window's runs, in US dollars. */
	costUsd: number;
	avgDurationMinutes: number;
	/** Processing volume over the window, oldest bucket first. */
	volume: { bucketHours: number; buckets: VolumeBucket[] };
}

export function isRunning(run: ProcessingRun): boolean {
	return run.result === null;
}

/** The run with `id`, else the newest one; undefined when there are none. */
export function selectRun(
	runs: readonly ProcessingRun[],
	id: string | null | undefined
): ProcessingRun | undefined {
	return runs.find((run) => run.id === id) ?? runs[0];
}

/** The newest run that has finished. */
export function latestFinishedRun(runs: readonly ProcessingRun[]): ProcessingRun | undefined {
	return runs.find((run) => !isRunning(run));
}

/** Whether some of a finished run's posts are still between stages. */
export function hasPendingPosts(result: RunResult): boolean {
	return PROCESSING_STATUSES.some(
		(status) => !isFinalStatus(status) && result.statusCounts[status] > 0
	);
}

/** Posts processed and failed over the whole window. */
export function volumeTotals(buckets: readonly VolumeBucket[]): {
	processed: number;
	failed: number;
} {
	return buckets.reduce(
		(sum, bucket) => ({
			processed: sum.processed + bucket.processed,
			failed: sum.failed + bucket.failed
		}),
		{ processed: 0, failed: 0 }
	);
}

/** Share of a bucket's posts processed successfully, in percent; null when empty. */
export function successRate({ processed, failed }: VolumeBucket): number | null {
	const total = processed + failed;
	return total === 0 ? null : (processed / total) * 100;
}

/** Average cost of one run; 0 when there were no runs. */
export function costPerRun(snapshot: Pick<MonitorSnapshot, 'costUsd' | 'runCount'>): number {
	return snapshot.runCount === 0 ? 0 : snapshot.costUsd / snapshot.runCount;
}
