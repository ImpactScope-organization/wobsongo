/**
 * A relative link to the same page with one search param changed and the
 * others kept, e.g. switching the time range keeps the topic.
 */
export function withSearchParam(url: Pick<URL, 'search'>, key: string, value: string): string {
	const params = new URLSearchParams(url.search);
	params.set(key, value);
	return `?${params}`;
}
