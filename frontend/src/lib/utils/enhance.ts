import type { ActionReturn } from 'svelte/action';

/**
 * Progressive enhancement hook for forms, e.g. SvelteKit's `enhance` from
 * `$app/forms`. Components accept it as an optional prop so they stay
 * presentational: without it, forms submit as plain HTML.
 */
export type FormEnhance = (form: HTMLFormElement) => ActionReturn | void;

/** `use:optionalEnhance={enhance}`: applies `enhance` when given, else does nothing. */
export function optionalEnhance(form: HTMLFormElement, enhance?: FormEnhance) {
	return enhance?.(form) ?? undefined;
}
