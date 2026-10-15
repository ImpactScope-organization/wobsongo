import { describe, expect, it } from 'vitest';
import { failure } from '../../result.js';
import { createApiRepos } from './index.js';

const BASE = 'https://api.example.org';

const tokens = {
	access_token: 'access-1',
	access_expires_at: '2026-10-15T09:15:00Z',
	refresh_token: 'refresh-1',
	refresh_expires_at: '2026-10-18T09:00:00Z'
};

const me = {
	id: '0199e6c4-0000-7000-8000-000000000001',
	name: 'Aminata Ouédraogo',
	email: 'a.ouedraogo@wobsongo.org',
	role: 'admin',
	password_changed_at: '2026-07-08T09:30:00Z'
};

interface Sent {
	method: string;
	path: string;
	authorization: string | null;
	body: unknown;
}

/** API repos over a fake backend that answers `status` and `body` to every call. */
function backend(status: number, body: unknown) {
	const sent: Sent[] = [];
	const fetch = async (input: RequestInfo | URL) => {
		const request = input as Request;
		const text = await request.text();
		sent.push({
			method: request.method,
			path: new URL(request.url).pathname,
			authorization: request.headers.get('authorization'),
			body: text ? JSON.parse(text) : undefined
		});
		return Response.json(body, { status });
	};
	return {
		sent,
		repos: createApiRepos({ baseUrl: BASE, fetch: fetch as typeof globalThis.fetch })
	};
}

const envelope = (data: unknown, status = 200) => ({ request_id: 'r', status, data });
const apiError = (status: number, error: string) => ({ request_id: 'r', status, error });
const scope = { accountId: me.id };

describe('API auth repo', () => {
	it('logs in and maps the token pair', async () => {
		const { sent, repos } = backend(200, envelope(tokens));
		const result = await repos.auth.login({ email: 'a@x.org', password: 'pw-123456789' });
		expect(result).toEqual({
			ok: true,
			data: {
				accessToken: 'access-1',
				accessExpiresAt: new Date('2026-10-15T09:15:00Z'),
				refreshToken: 'refresh-1',
				refreshExpiresAt: new Date('2026-10-18T09:00:00Z')
			}
		});
		expect(sent[0]).toMatchObject({
			method: 'POST',
			path: '/api/v1/auth/login',
			body: { email: 'a@x.org', password: 'pw-123456789' }
		});
	});

	it("keeps the backend's status and message on failure", async () => {
		const { repos } = backend(401, apiError(401, 'incorrect email or password'));
		expect(await repos.auth.login({ email: 'a@x.org', password: 'nope' })).toEqual(
			failure(401, 'incorrect email or password')
		);
	});

	it('signs up and refreshes', async () => {
		const { sent, repos } = backend(201, envelope(tokens, 201));
		expect((await repos.auth.signup({ name: 'A', email: 'a@x.org', password: 'p' })).ok).toBe(true);
		expect((await repos.auth.refresh('refresh-0')).ok).toBe(true);
		expect(sent.map(({ path }) => path)).toEqual(['/api/v1/auth/signup', '/api/v1/auth/refresh']);
		expect(sent[1].body).toEqual({ refresh_token: 'refresh-0' });
	});

	it('reads the session with the given access token', async () => {
		const { sent, repos } = backend(200, envelope(me));
		expect(await repos.auth.getSession('access-9')).toEqual({
			ok: true,
			data: { scope: { accountId: me.id }, name: me.name }
		});
		expect(sent[0]).toMatchObject({ path: '/api/v1/me', authorization: 'Bearer access-9' });
	});

	it('logs out without calling the backend: its tokens are stateless', async () => {
		const { sent, repos } = backend(500, {});
		expect((await repos.auth.logout('refresh-1')).ok).toBe(true);
		expect(sent).toEqual([]);
	});

	it('reports an unreachable backend as an outage, not a rejected token', async () => {
		const repos = createApiRepos({
			baseUrl: BASE,
			fetch: () => Promise.reject(new TypeError('fetch failed'))
		});
		expect((await repos.auth.getSession('access-1')).ok).toBe(false);
		expect(await repos.auth.getSession('access-1')).toMatchObject({ status: 503 });
	});

	it('treats a success without data as a bad response', async () => {
		const { repos } = backend(200, { request_id: 'r', status: 200 });
		expect(await repos.auth.refresh('refresh-1')).toMatchObject({ ok: false, status: 502 });
	});
});

describe('API account repo', () => {
	it('maps the settings', async () => {
		const { repos } = backend(200, envelope(me));
		expect(await repos.account.getSettings(scope)).toEqual({
			ok: true,
			data: {
				profile: { name: me.name, email: me.email, role: 'admin' },
				passwordChangedAt: new Date('2026-07-08T09:30:00Z')
			}
		});
	});

	it('updates the profile', async () => {
		const { sent, repos } = backend(200, envelope({ ...me, name: 'A. O.' }));
		const result = await repos.account.updateProfile(scope, { name: 'A. O.', email: me.email });
		expect(result.ok && result.data.name).toBe('A. O.');
		expect(sent[0]).toMatchObject({
			method: 'PATCH',
			path: '/api/v1/me',
			body: { name: 'A. O.', email: me.email }
		});
	});

	it('changes the password and returns the new tokens', async () => {
		const { sent, repos } = backend(200, envelope(tokens));
		const result = await repos.account.changePassword(scope, {
			currentPassword: 'old-password-1',
			newPassword: 'new-password-12'
		});
		expect(result.ok && result.data.refreshToken).toBe('refresh-1');
		expect(sent[0]).toMatchObject({
			path: '/api/v1/me/password',
			body: { current_password: 'old-password-1', new_password: 'new-password-12' }
		});
	});

	it('maps a wrong current password to 403', async () => {
		const { repos } = backend(403, apiError(403, 'current password is incorrect'));
		const result = await repos.account.changePassword(scope, {
			currentPassword: 'x',
			newPassword: 'new-password-12'
		});
		expect(result).toMatchObject({ ok: false, status: 403 });
	});

	it('has no password reset yet', async () => {
		const { repos } = backend(200, {});
		expect(await repos.account.requestPasswordReset(scope)).toEqual(
			failure(501, 'Password reset is not available yet.')
		);
	});
});
