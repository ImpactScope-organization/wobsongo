import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	API_HOST: {
		description:
			'Where data comes from: `stub://local` for the built-in stub data, or the backend URL, e.g. `https://api.wobsongo.org`.',
		// Optional here so `svelte-kit sync` and tests run without it; the server
		// refuses to start without it (hooks.server.ts).
		schema: (value) => value || undefined
	}
});
