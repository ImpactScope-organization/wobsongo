/** Reporting windows offered on the Overview view. */
export type TimeRange = '24h' | '7d' | '30d';

export const TIME_RANGES: readonly TimeRange[] = ['24h', '7d', '30d'];

export const DEFAULT_TIME_RANGE: TimeRange = '24h';

export function parseTimeRange(value: string | null | undefined): TimeRange {
	return TIME_RANGES.find((range) => range === value) ?? DEFAULT_TIME_RANGE;
}
