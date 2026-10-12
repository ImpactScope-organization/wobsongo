/**
 * A count that may be withheld for privacy. Counts below the minimum cell size
 * are suppressed on the server, so small numbers that could identify people
 * never reach the browser.
 */
export type Count = { kind: 'value'; value: number } | { kind: 'suppressed'; threshold: number };

/** Default minimum cell size: counts under this are suppressed. */
export const DEFAULT_MIN_CELL_SIZE = 10;

export function toCount(value: number, minCellSize = DEFAULT_MIN_CELL_SIZE): Count {
	return value < minCellSize
		? { kind: 'suppressed', threshold: minCellSize }
		: { kind: 'value', value };
}
