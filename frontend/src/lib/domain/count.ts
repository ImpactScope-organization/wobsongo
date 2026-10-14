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

/**
 * A total shown next to counts that may be suppressed. When any part is
 * suppressed the exact total would give it away (total minus the visible
 * parts), so it is rounded to the minimum cell size instead.
 */
export interface Total {
	value: number;
	approximate: boolean;
}

export function toTotal(
	value: number,
	parts: readonly Count[],
	minCellSize = DEFAULT_MIN_CELL_SIZE
): Total {
	return parts.some((part) => part.kind === 'suppressed')
		? { value: Math.round(value / minCellSize) * minCellSize, approximate: true }
		: { value, approximate: false };
}
