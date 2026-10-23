import { defineConfig } from '@inlang/paraglide-js';

// Read by both the Vite plugin and `paraglide-js compile` (pnpm prepare/check),
// so dev, tests and type checks get the same runtime.
export default defineConfig({
	outdir: './src/lib/paraglide',
	emitTsDeclarations: true,
	// No locale in URLs: the choice lives in a cookie, set by POST /lang.
	// Without one, French (the base locale).
	strategy: ['cookie', 'globalVariable', 'baseLocale'],
	cookieName: 'wobsongo_lang',
	cookieSameSite: 'lax'
});
