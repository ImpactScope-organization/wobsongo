/**
 * Sample data for the Monitor view, taken from the Infodemic Dashboard design.
 * Shared by stories, component tests and the stub repositories, so
 * treat it as read-only.
 */
import {
	CRAWLED_PLATFORMS,
	type CrawledPlatform,
	type MonitorSnapshot,
	type ProcessingRun,
	type ProcessingStatus,
	type RunResult,
	type VolumeBucket
} from '#lib/domain/monitor.js';
import type { TimeRange } from '#lib/domain/time-range.js';
import { OVERVIEW_NOW } from './overview.js';

/** "Now" for every Monitor fixture, the same as Overview's: 12 Oct 2026, 14:05 GMT. */
export const MONITOR_NOW = OVERVIEW_NOW;

const HOUR = 60 * 60 * 1000;
const DAY = 24 * HOUR;

/** Posts still between stages in a run that has only just finished. */
const STILL_PROCESSING = {
	discovered: 3,
	screened_in: 4,
	transcribed: 5,
	analyzed: 3,
	verified: 2
} satisfies Partial<Record<ProcessingStatus, number>>;

/** Where a run's new posts ended up: most are screened out or clustered. */
function statusCounts(
	newPosts: number,
	failed: number,
	stillProcessing: boolean
): Record<ProcessingStatus, number> {
	const pending: Partial<Record<ProcessingStatus, number>> = stillProcessing
		? STILL_PROCESSING
		: {};
	const pendingTotal = Object.values(pending).reduce((sum, count) => sum + count, 0);
	const screenedOut = Math.round(newPosts * 0.38);
	const outOfScope = Math.round((newPosts - screenedOut) * 0.11);
	return {
		discovered: pending.discovered ?? 0,
		screened_out: screenedOut,
		screened_in: pending.screened_in ?? 0,
		transcribed: pending.transcribed ?? 0,
		out_of_scope: outOfScope,
		analyzed: pending.analyzed ?? 0,
		verified: pending.verified ?? 0,
		clustered: newPosts - screenedOut - outOfScope - failed - pendingTotal,
		failed
	};
}

const PLATFORM_WEIGHT: Record<CrawledPlatform, number> = {
	facebook: 34,
	tiktok: 29,
	x: 14,
	instagram: 13,
	youtube: 10
};

/** New posts split by platform, varied a little per run; sums to `newPosts`. */
function platformCounts(newPosts: number, index: number): Record<CrawledPlatform, number> {
	const weights = CRAWLED_PLATFORMS.map((platform, j) =>
		Math.max(2, PLATFORM_WEIGHT[platform] + ((index * 3 + j * 5) % 7) - 3)
	);
	const total = weights.reduce((sum, weight) => sum + weight, 0);
	const counts = weights.map((weight) => Math.floor((newPosts * weight) / total));
	counts[0] += newPosts - counts.reduce((sum, count) => sum + count, 0);
	return Object.fromEntries(
		CRAWLED_PLATFORMS.map((platform, j) => [platform, counts[j]])
	) as Record<CrawledPlatform, number>;
}

type RunRow = [
	startedAt: string,
	executedBy: string | null,
	discovered: number,
	newPosts: number,
	durationMinutes: number,
	failed: number,
	costUsd: number
];

// Newest first. The 14:00 run has finished, but some of its posts are still being processed.
const RUN_ROWS: RunRow[] = [
	['2026-10-12T14:00:00Z', null, 412, 138, 23, 3, 2.84],
	['2026-10-12T12:00:00Z', null, 398, 121, 21, 1, 2.61],
	['2026-10-12T11:20:00Z', 'Admin 1', 156, 44, 9, 0, 1.02],
	['2026-10-12T10:00:00Z', null, 436, 152, 26, 5, 3.1],
	['2026-10-12T08:00:00Z', null, 371, 98, 19, 2, 2.33],
	['2026-10-12T06:00:00Z', null, 288, 77, 15, 4, 1.94],
	['2026-10-12T04:00:00Z', null, 214, 52, 12, 0, 1.41],
	['2026-10-12T02:00:00Z', null, 236, 61, 13, 1, 1.55],
	['2026-10-12T00:00:00Z', null, 301, 85, 16, 2, 1.98],
	['2026-10-11T22:41:00Z', 'Admin 2', 122, 30, 7, 0, 0.81],
	['2026-10-11T22:00:00Z', null, 349, 104, 18, 3, 2.27]
];

export const processingRuns: ProcessingRun[] = RUN_ROWS.map(
	([startedAt, executedBy, discovered, newPosts, durationMinutes, failed, costUsd], index) => {
		const result: RunResult = {
			discovered,
			newPosts,
			durationMinutes,
			failed,
			costUsd,
			statusCounts: statusCounts(newPosts, failed, index === 0),
			platformCounts: platformCounts(newPosts, index)
		};
		const at = new Date(startedAt);
		return {
			id: `run-${startedAt.slice(0, 16).replace(/[-:T]/g, '')}`,
			startedAt: at,
			type: executedBy ? 'manual' : 'automatic',
			executedBy,
			result
		};
	}
);

/** A manual run that has just been started. */
export const runningRun: ProcessingRun = {
	id: 'run-202610121407',
	startedAt: new Date('2026-10-12T14:07:00Z'),
	type: 'manual',
	executedBy: 'Admin 1',
	result: null
};

/** Buckets of `hours`, the last one ending at the window's end. */
function buckets(
	end: Date,
	hours: number,
	processed: readonly number[],
	failed: readonly number[]
): VolumeBucket[] {
	const first = end.getTime() - processed.length * hours * HOUR;
	return processed.map((count, index) => ({
		start: new Date(first + index * hours * HOUR),
		processed: count,
		failed: failed[index]
	}));
}

const LAST_RUN_END = new Date('2026-10-12T14:00:00Z');
const TODAY_END = new Date(MONITOR_NOW.getTime() - (MONITOR_NOW.getTime() % DAY) + DAY);

/** Window-wide figures for each time range. */
export const monitorWindows: Record<
	TimeRange,
	Pick<MonitorSnapshot, 'runCount' | 'costUsd' | 'avgDurationMinutes' | 'volume'>
> = {
	'24h': {
		runCount: 12,
		costUsd: 24.86,
		avgDurationMinutes: 17,
		volume: {
			bucketHours: 2,
			buckets: buckets(
				LAST_RUN_END,
				2,
				[62, 41, 58, 39, 51, 33, 72, 64, 88, 79, 91, 84],
				[1, 0, 2, 1, 1, 4, 0, 1, 5, 2, 1, 3]
			)
		}
	},
	'7d': {
		runCount: 86,
		costUsd: 171.2,
		avgDurationMinutes: 18,
		volume: {
			bucketHours: 24,
			buckets: buckets(
				TODAY_END,
				24,
				[1012, 1104, 968, 1180, 1045, 1017, 1086],
				[18, 22, 15, 30, 19, 13, 21]
			)
		}
	},
	'30d': {
		runCount: 368,
		costUsd: 742.55,
		avgDurationMinutes: 18,
		volume: {
			bucketHours: 24,
			buckets: buckets(
				TODAY_END,
				24,
				Array.from({ length: 30 }, (_, i) => 920 + Math.round(Math.sin(i * 1.3) * 90 + i * 6)),
				Array.from({ length: 30 }, (_, i) => 12 + ((i * 7) % 15))
			)
		}
	}
};

export const monitorSnapshot: MonitorSnapshot = {
	updatedAt: MONITOR_NOW,
	runs: processingRuns,
	...monitorWindows['24h']
};
