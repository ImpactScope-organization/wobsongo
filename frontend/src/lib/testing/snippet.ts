import { createRawSnippet } from 'svelte';

/** A snippet that renders fixed HTML, for passing `children` and friends in SSR tests. */
export function htmlSnippet(html: string) {
	return createRawSnippet(() => ({ render: () => html }));
}

/** SSR output without Svelte's hydration comments, for readable assertions. */
export function stripComments(html: string): string {
	return html.replace(/<!--[\s\S]*?-->/g, '');
}
