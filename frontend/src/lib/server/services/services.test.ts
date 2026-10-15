import { describe, expect, it } from 'vitest';
import { PULSE_NOW } from '#lib/fixtures/pulse.js';
import { profile } from '#lib/fixtures/settings.js';
import { failure } from '../result.js';
import { createApiRepos } from '../repos/api/index.js';
import { ACCESS_TOKEN_TTL_MS, REFRESH_TOKEN_TTL_MS } from '../repos/stub/auth.js';
import {
	DEMO_ACCOUNT_ID,
	DEMO_EMAIL,
	DEMO_PASSWORD,
	DEMO_TENANT_ID,
	ERROR_EMAIL_DOMAIN
} from '../repos/stub/index.js';
import { createTestBackend } from '../testing/fakes.js';
import { describeFailure, SERVER_ERROR_MESSAGE } from './failures.js';
import { createServices } from './index.js';
import { PULSE_NARRATIVE_LIMIT } from './pulse.js';

const demo = { tenantId: DEMO_TENANT_ID, accountId: DEMO_ACCOUNT_ID };
const apiServices = createServices(
	createApiRepos({ baseUrl: 'https://api.example.org', fetch: () => Promise.reject() })
);

const newAccount = {
	name: 'Kadi Sawadogo',
	email: 'k.sawadogo@wobsongo.org',
	password: 'kadi-password-1',
	confirmPassword: 'kadi-password-1'
};

describe('describeFailure', () => {
	it('words expected statuses, hides server errors and keeps "not available yet"', () => {
		expect(describeFailure(failure(409, 'raw'), { 409: 'Taken.' }).message).toBe('Taken.');
		expect(describeFailure(failure(503, 'upstream timeout')).message).toBe(SERVER_ERROR_MESSAGE);
		expect(describeFailure(failure(501, 'Pulse is not available yet.')).message).toBe(
			'Pulse is not available yet.'
		);
		expect(describeFailure(failure(400, 'Bad input.')).message).toBe('Bad input.');
	});
});

describe('PulseService', () => {
	it('shows misinformation narratives, highest volume first', async () => {
		const { services } = createTestBackend();
		const result = await services.pulse.getPulse(demo, { range: '24h', topic: 'misinfo' });
		if (!result.ok) throw new Error(result.message);

		const { narratives } = result.data;
		expect(narratives.every((narrative) => narrative.verdict !== 'unverified')).toBe(true);
		expect(narratives.map((narrative) => narrative.volume)).toEqual(
			narratives.map((narrative) => narrative.volume).toSorted((a, b) => b - a)
		);
		expect(narratives.length).toBeLessThanOrEqual(PULSE_NARRATIVE_LIMIT);
	});

	it('adds unverified narratives for all topics', async () => {
		const { services } = createTestBackend();
		const result = await services.pulse.getPulse(demo, { range: '24h', topic: 'all' });
		expect(result.ok && result.data.narratives.some((n) => n.verdict === 'unverified')).toBe(true);
	});

	it('stamps the snapshot with the clock', async () => {
		const { clock, services } = createTestBackend();
		clock.advance(60_000);
		const result = await services.pulse.getPulse(demo, { range: '24h', topic: 'misinfo' });
		expect(result.ok && result.data.updatedAt).toEqual(new Date(PULSE_NOW.getTime() + 60_000));
	});

	it('scales longer windows in the stub, without the anomaly banner', async () => {
		const { services } = createTestBackend();
		const day = await services.pulse.getPulse(demo, { range: '24h', topic: 'misinfo' });
		const week = await services.pulse.getPulse(demo, { range: '7d', topic: 'misinfo' });
		if (!day.ok || !week.ok) throw new Error('expected data');
		expect(week.data.messages.current).toBeGreaterThan(day.data.messages.current);
		expect(day.data.anomaly).not.toBeNull();
		expect(week.data.anomaly).toBeNull();
	});

	it('follows the stub id conventions for the tenant', async () => {
		const { services } = createTestBackend();
		const query = { range: '24h', topic: 'misinfo' } as const;
		expect(await services.pulse.getPulse({ ...demo, tenantId: 't-404' }, query)).toEqual(
			failure(404, 'No Pulse data for this account.')
		);
		expect(await services.pulse.getPulse({ ...demo, tenantId: 't-500' }, query)).toEqual(
			failure(500, SERVER_ERROR_MESSAGE)
		);
	});

	it('is not available yet on the backend', async () => {
		expect(await apiServices.pulse.getPulse(demo, { range: '24h', topic: 'misinfo' })).toEqual(
			failure(501, 'Pulse is not available yet.')
		);
	});
});

describe('AccountService', () => {
	it('returns the settings', async () => {
		const { services } = createTestBackend();
		const result = await services.account.getSettings(demo);
		expect(result.ok && result.data.profile).toEqual(profile);
	});

	it('follows the stub id conventions for the account', async () => {
		const { services } = createTestBackend();
		expect(await services.account.getSettings({ ...demo, accountId: 'acc-404' })).toEqual(
			failure(404, 'Your account was not found.')
		);
	});

	describe('updateProfile', () => {
		it('saves a trimmed name and normalized email', async () => {
			const { services } = createTestBackend();
			const result = await services.account.updateProfile(demo, {
				name: '  Aminata Ouédraogo ',
				email: ' Aminata@WobSongo.org'
			});
			expect(result.ok && result.data).toMatchObject({
				name: 'Aminata Ouédraogo',
				email: 'aminata@wobsongo.org'
			});

			const settings = await services.account.getSettings(demo);
			expect(settings.ok && settings.data.profile.email).toBe('aminata@wobsongo.org');
		});

		it('rejects invalid input without calling the repository', async () => {
			const { services } = createTestBackend();
			expect(await services.account.updateProfile(demo, { name: '', email: 'x@y.org' })).toEqual(
				failure(400, 'Enter your name.')
			);
		});

		it('rejects an email another account uses', async () => {
			const { services } = createTestBackend();
			await services.auth.signup(newAccount);
			const result = await services.account.updateProfile(demo, {
				name: profile.name,
				email: newAccount.email
			});
			expect(result).toEqual(failure(409, 'Another account already uses this email.'));
		});

		it('hides server errors', async () => {
			const { services } = createTestBackend();
			const result = await services.account.updateProfile(demo, {
				name: profile.name,
				email: `down${ERROR_EMAIL_DOMAIN}`
			});
			expect(result).toEqual(failure(500, SERVER_ERROR_MESSAGE));
		});
	});

	describe('changePassword', () => {
		const change = {
			currentPassword: DEMO_PASSWORD,
			newPassword: 'a-brand-new-password',
			confirmPassword: 'a-brand-new-password'
		};

		it('changes it, records when, and the new one signs in', async () => {
			const { clock, services } = createTestBackend();
			clock.advance(5_000);
			expect((await services.account.changePassword(demo, change)).ok).toBe(true);

			const settings = await services.account.getSettings(demo);
			expect(settings.ok && settings.data.passwordChangedAt).toEqual(clock.now());
			const login = await services.auth.login({ email: DEMO_EMAIL, password: change.newPassword });
			expect(login.ok).toBe(true);
		});

		it('rejects a wrong current password', async () => {
			const { services } = createTestBackend();
			const result = await services.account.changePassword(demo, {
				...change,
				currentPassword: 'not-my-password'
			});
			expect(result).toEqual(failure(403, 'Your current password is incorrect.'));
		});

		it('validates before calling the repository', async () => {
			const { services } = createTestBackend();
			const result = await services.account.changePassword(demo, {
				...change,
				confirmPassword: 'something-else-entirely'
			});
			expect(result).toEqual(failure(400, 'The passwords don’t match.'));
		});
	});

	it('requests a password reset for the account email', async () => {
		const { services } = createTestBackend();
		const result = await services.account.requestPasswordReset(demo);
		expect(result.ok && result.data).toEqual({ sentTo: profile.email });
	});

	it('is not available yet on the backend', async () => {
		expect(await apiServices.account.getSettings(demo)).toEqual(
			failure(501, 'Settings is not available yet.')
		);
	});
});

describe('AuthService', () => {
	describe('login', () => {
		it('signs the demo reviewer in, whatever the email case', async () => {
			const { services } = createTestBackend();
			const tokens = await services.auth.login({
				email: DEMO_EMAIL.toUpperCase(),
				password: DEMO_PASSWORD
			});
			if (!tokens.ok) throw new Error(tokens.message);

			const session = await services.auth.getSession(tokens.data.accessToken);
			expect(session.ok && session.data).toEqual({ scope: demo, name: profile.name });
		});

		it('gives the same answer for a wrong password and an unknown email', async () => {
			const { services } = createTestBackend();
			const wrong = await services.auth.login({ email: DEMO_EMAIL, password: 'nope-nope-nope' });
			const unknown = await services.auth.login({
				email: 'nobody@wobsongo.org',
				password: DEMO_PASSWORD
			});
			expect(wrong).toEqual(failure(401, 'Email or password is incorrect.'));
			expect(unknown).toEqual(wrong);
		});

		it('asks for both fields', async () => {
			const { services } = createTestBackend();
			expect(await services.auth.login({ email: ' ', password: '' })).toEqual(
				failure(400, 'Enter your email and password.')
			);
		});

		it('is not available yet on the backend', async () => {
			expect(await apiServices.auth.login({ email: DEMO_EMAIL, password: DEMO_PASSWORD })).toEqual(
				failure(501, 'Logging in is not available yet.')
			);
		});
	});

	describe('signup', () => {
		it('creates an account in a tenant of its own, signed in', async () => {
			const { services } = createTestBackend();
			const tokens = await services.auth.signup(newAccount);
			if (!tokens.ok) throw new Error(tokens.message);

			const session = await services.auth.getSession(tokens.data.accessToken);
			if (!session.ok) throw new Error(session.message);
			expect(session.data.name).toBe(newAccount.name);
			expect(session.data.scope.tenantId).not.toBe(DEMO_TENANT_ID);

			const settings = await services.account.getSettings(session.data.scope);
			expect(settings.ok && settings.data.profile.email).toBe(newAccount.email);
		});

		it('rejects a taken email', async () => {
			const { services } = createTestBackend();
			const result = await services.auth.signup({ ...newAccount, email: DEMO_EMAIL });
			expect(result).toEqual(failure(409, 'An account with this email already exists.'));
		});

		it('validates the name, email and password', async () => {
			const { services } = createTestBackend();
			expect(await services.auth.signup({ ...newAccount, email: 'kadi' })).toEqual(
				failure(400, 'Enter a valid email address.')
			);
			expect(
				await services.auth.signup({ ...newAccount, password: 'short', confirmPassword: 'short' })
			).toEqual(failure(400, 'Use at least 12 characters for your password.'));
		});
	});

	describe('tokens', () => {
		async function signedIn() {
			const backend = createTestBackend();
			const tokens = await backend.services.auth.login({
				email: DEMO_EMAIL,
				password: DEMO_PASSWORD
			});
			if (!tokens.ok) throw new Error(tokens.message);
			return { ...backend, tokens: tokens.data };
		}

		it('expire the access token after 15 minutes', async () => {
			const { clock, services, tokens } = await signedIn();
			expect(tokens.accessExpiresAt).toEqual(new Date(PULSE_NOW.getTime() + ACCESS_TOKEN_TTL_MS));

			clock.advance(ACCESS_TOKEN_TTL_MS);
			expect(await services.auth.getSession(tokens.accessToken)).toEqual(
				failure(401, 'Not signed in.')
			);
		});

		it('refresh a new access token with the refresh token', async () => {
			const { clock, services, tokens } = await signedIn();
			clock.advance(ACCESS_TOKEN_TTL_MS);

			const refreshed = await services.auth.refresh(tokens.refreshToken);
			if (!refreshed.ok) throw new Error(refreshed.message);
			expect(refreshed.data.accessToken).not.toBe(tokens.accessToken);
			expect((await services.auth.getSession(refreshed.data.accessToken)).ok).toBe(true);
		});

		it('stop refreshing after 7 days', async () => {
			const { clock, services, tokens } = await signedIn();
			clock.advance(REFRESH_TOKEN_TTL_MS);
			expect((await services.auth.refresh(tokens.refreshToken)).ok).toBe(false);
		});

		it('are both revoked on logout', async () => {
			const { services, tokens } = await signedIn();
			await services.auth.logout(tokens.refreshToken);
			expect((await services.auth.getSession(tokens.accessToken)).ok).toBe(false);
			expect((await services.auth.refresh(tokens.refreshToken)).ok).toBe(false);
		});
	});
});
