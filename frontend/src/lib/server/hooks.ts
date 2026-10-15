import { redirect } from '@sveltejs/kit';
import type { Handle, HandleFetch } from '@sveltejs/kit/hooks';
import type { Clock } from '#lib/utils/clock.js';
import { ACCESS_COOKIE } from './auth/cookies.js';
import { loginUrl, resolveSession } from './auth/session.js';
import type { Services } from './services/index.js';

/** Pages for signing in, which signed-in people skip. */
const AUTH_ROUTES = new Set(['/login', '/signup']);

/** `app` routes (the `(app)` group) need a session; `auth` routes are for getting one. */
export function routeGroup(routeId: string | null): 'app' | 'auth' | null {
	if (routeId?.startsWith('/(app)')) return 'app';
	if (routeId && AUTH_ROUTES.has(routeId)) return 'auth';
	return null;
}

export interface HandleOptions {
	clock: Clock;
	servicesFor: (fetch: typeof globalThis.fetch) => Services;
}

/**
 * Sets up `locals` and guards routes. The guard lives here, not only in the
 * `(app)` layout: page loads run in parallel with layout loads, and form
 * actions skip them, so a layout-only guard would leave both open.
 */
export function createHandle({ clock, servicesFor }: HandleOptions): Handle {
	return async ({ event, resolve }) => {
		const services = servicesFor(event.fetch);
		const session = await resolveSession(event.cookies, services.auth, clock);
		event.locals.clock = clock;
		event.locals.services = services;
		event.locals.session = session;

		const group = routeGroup(event.route.id);
		if (group === 'app' && !session) redirect(303, loginUrl(event.url));
		if (group === 'auth' && session) redirect(303, '/');

		return resolve(event);
	};
}

/** Sends the access token with requests to the backend, and only there. */
export function createHandleFetch(baseUrl: string | null): HandleFetch {
	return ({ event, request, fetch }) => {
		const token = event.cookies.get(ACCESS_COOKIE);
		if (baseUrl && token && request.url.startsWith(`${baseUrl}/`)) {
			request.headers.set('authorization', `Bearer ${token}`);
		}
		return fetch(request);
	};
}
