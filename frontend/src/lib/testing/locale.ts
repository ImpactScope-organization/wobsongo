import { beforeEach } from 'vitest';
import { setLocale, type Locale } from '#lib/paraglide/runtime.js';

/**
 * Server tests read English, whatever the app's default. Outside a request
 * Paraglide falls back to a global, which `withLocale` switches and every
 * test resets.
 */
export const TEST_LOCALE: Locale = 'en';

beforeEach(() => {
	setLocale(TEST_LOCALE, { reload: false });
});

/**
 * Runs `run` in another language, e.g. to check the French copy. Read what
 * you need inside it: `render(...).body` is built lazily, on first access.
 */
export function withLocale<T>(locale: Locale, run: () => T): T {
	setLocale(locale, { reload: false });
	try {
		return run();
	} finally {
		setLocale(TEST_LOCALE, { reload: false });
	}
}
