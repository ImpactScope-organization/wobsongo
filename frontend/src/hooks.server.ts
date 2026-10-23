import { getTextDirection } from '#lib/paraglide/runtime.js';
import { paraglideMiddleware } from '#lib/paraglide/server.js';
import { API_HOST } from '$app/env/private';
import { type Handle, type HandleFetch, type ServerInit, sequence } from '@sveltejs/kit/hooks';
import { parseApiHost, servicesFactory, type Backend } from '#lib/server/backend.js';
import { createHandle, createHandleFetch } from '#lib/server/hooks.js';
import { systemClock } from '#lib/utils/clock.js';

let handleRequest: Handle | undefined;
let handleBackendFetch: HandleFetch | undefined;

// Fails startup, not the first request, when API_HOST is missing or wrong.
export const init: ServerInit = () => {
	const backend: Backend = parseApiHost(API_HOST);
	if (backend.kind === 'stub')
		console.info('[backend] running on stub data (API_HOST=%s)', API_HOST);

	handleRequest = createHandle({
		clock: systemClock,
		servicesFor: servicesFactory(backend, systemClock)
	});
	handleBackendFetch = createHandleFetch(backend.kind === 'api' ? backend.baseUrl : null);
};

// The locale comes from the cookie (French when absent). It is set first so
// everything after it, including service messages, runs in that language.
const handleLocale: Handle = ({ event, resolve }) =>
	paraglideMiddleware(event.request, ({ locale }) =>
		resolve(event, {
			transformPageChunk: ({ html }) =>
				html
					.replace('%paraglide.lang%', locale)
					.replace('%paraglide.dir%', getTextDirection(locale))
		})
	);

const handleApp: Handle = (input) => {
	if (!handleRequest) throw new Error('Server hooks used before init.');
	return handleRequest(input);
};

export const handle = sequence(handleLocale, handleApp);

export const handleFetch: HandleFetch = (input) =>
	handleBackendFetch ? handleBackendFetch(input) : input.fetch(input.request);
