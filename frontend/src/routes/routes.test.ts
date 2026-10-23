import { isActionFailure, isHttpError, isRedirect } from '@sveltejs/kit';
import { assert, describe, expect, it } from 'vitest';
import { profile } from '#lib/fixtures/settings.js';
import { ACCESS_COOKIE, REFRESH_COOKIE } from '#lib/server/auth/cookies.js';
import { DEMO_EMAIL, DEMO_PASSWORD } from '#lib/server/repos/stub/index.js';
import {
	createFakeCookies,
	createTestBackend,
	formRequest,
	signInDemo,
	testLocals,
	type TestBackend
} from '#lib/server/testing/fakes.js';
import { load as layoutLoad } from './(app)/+layout.server';
import { load as overviewLoad } from './(app)/+page.server';
import { load as comingSoonLoad } from './(app)/[...path]/+page.server';
import { actions as settingsActions, load as settingsLoad } from './(app)/settings/+page.server';
import { actions as loginActions } from './login/+page.server';
import { POST as setLanguage } from './lang/+server';
import { POST as logout } from './logout/+server';
import { actions as signupActions } from './signup/+page.server';

const ORIGIN = 'https://dashboard.example';

/** Loads and actions only read what they use; the rest of the event is left out. */
function event<T>(fields: Record<string, unknown>): T {
	return fields as T;
}

/** What a thrown redirect or error carries, or the returned value. */
async function settle(run: () => unknown) {
	try {
		return { value: await run() };
	} catch (thrown) {
		if (isRedirect(thrown)) return { redirect: thrown.location };
		if (isHttpError(thrown)) return { error: thrown.status, message: thrown.body.message };
		throw thrown;
	}
}

async function signedIn(backend: TestBackend = createTestBackend()) {
	const { cookies, session } = await signInDemo(backend);
	return { backend, cookies, locals: testLocals(backend, session) };
}

describe('(app) layout', () => {
	it('gives the header the signed-in name', async () => {
		const { locals } = await signedIn();
		const data = await layoutLoad(event({ locals, url: new URL(ORIGIN) }));
		expect(data).toEqual({ account: { name: profile.name } });
	});

	it('sends signed-out requests to login', async () => {
		const backend = createTestBackend();
		const result = await settle(() =>
			layoutLoad(event({ locals: testLocals(backend, null), url: new URL('/settings', ORIGIN) }))
		);
		expect(result).toEqual({ redirect: '/login?redirectTo=%2Fsettings' });
	});
});

describe('Overview page', () => {
	const loadAt = async (path: string, backend?: TestBackend) => {
		const { locals } = await signedIn(backend);
		return settle(() => overviewLoad(event({ locals, url: new URL(path, ORIGIN) })));
	};

	it('defaults to the last 24 h of misinformation', async () => {
		const { value } = await loadAt('/');
		expect(value).toMatchObject({ range: '24h', topic: 'misinfo' });
	});

	it('reads the range and topic from the URL', async () => {
		const { value } = await loadAt('/?range=7d&topic=all');
		expect(value).toMatchObject({ range: '7d', topic: 'all' });
	});

	it('ignores unknown values', async () => {
		const { value } = await loadAt('/?range=1y&topic=gossip');
		expect(value).toMatchObject({ range: '24h', topic: 'misinfo' });
	});

	it('shows service failures as error pages', async () => {
		const backend = createTestBackend();
		const { locals } = await signedIn(backend);
		const result = await settle(() =>
			overviewLoad(
				event({
					locals: {
						...locals,
						session: { ...locals.session!, scope: { accountId: 'acc-500' } }
					},
					url: new URL('/', ORIGIN)
				})
			)
		);
		expect(result).toMatchObject({ error: 500 });
	});

	it('filters creator names to pseudonyms and masks stats by default for non-whitelisted users, even with privacy=false', async () => {
		delete process.env.PRIVACY_WHITELIST_EMAILS;

		const { value } = await loadAt('/');
		const snapshot = (value as { snapshot: { creators?: Array<Record<string, unknown>> } })
			.snapshot;
		expect(snapshot.creators).toBeDefined();
		expect(snapshot.creators!.length).toBeGreaterThan(0);
		for (const creator of snapshot.creators!) {
			expect(creator.followers).toBeUndefined();
			expect(creator.postsCount).toBeUndefined();
			expect(creator.handle).toBeUndefined();
			expect(creator.profileUrl).toBeUndefined();
			expect(creator.name).toMatch(/^Creator \d+$/);
			expect(creator.name).not.toBe('Ibrahim Cissé');
			expect(creator.name).not.toBe('Aïcha Ouédraogo');
		}

		// Non-whitelisted user cannot bypass privacy mode with privacy=false
		const nonWhitelistedAttempt = await loadAt('/?privacy=false');
		const attemptSnapshot = (
			nonWhitelistedAttempt.value as { snapshot: { creators?: Array<Record<string, unknown>> } }
		).snapshot;
		for (const creator of attemptSnapshot.creators!) {
			expect(creator.followers).toBeUndefined();
			expect(creator.postsCount).toBeUndefined();
			expect(creator.handle).toBeUndefined();
		}

		const noAccess = await loadAt('/?hasAccess=false');
		const noAccessSnapshot = (
			noAccess.value as { snapshot: { creators?: Array<Record<string, unknown>> } }
		).snapshot;
		for (const creator of noAccessSnapshot.creators!) {
			expect(creator.followers).toBeUndefined();
			expect(creator.postsCount).toBeUndefined();
			expect(creator.handle).toBeUndefined();
			expect(creator.profileUrl).toBeUndefined();
		}
	});

	it('automatically reveals full creator data for whitelisted users, with optional privacy=true toggle', async () => {
		process.env.PRIVACY_WHITELIST_EMAILS = 'a.ouedraogo@wobsongo.org, other@wobsongo.org';
		try {
			// Whitelisted user automatically sees full data on "/"
			const { value } = await loadAt('/');
			const snapshot = (value as { snapshot: { creators?: Array<Record<string, unknown>> } })
				.snapshot;
			expect(snapshot.creators).toBeDefined();
			expect(snapshot.creators!.length).toBeGreaterThan(0);
			const first = snapshot.creators![0];
			expect(first.name).toBe('Ibrahim Cissé');
			expect(first.handle).toBe('@ibrahim_cisse');
			expect(first.followers).toBe(482000);
			expect(first.postsCount).toBe(38);

			// Whitelisted user can still toggle private mode on with ?privacy=true
			const privateView = await loadAt('/?privacy=true');
			const privateSnapshot = (
				privateView.value as { snapshot: { creators?: Array<Record<string, unknown>> } }
			).snapshot;
			for (const creator of privateSnapshot.creators!) {
				expect(creator.followers).toBeUndefined();
				expect(creator.postsCount).toBeUndefined();
				expect(creator.handle).toBeUndefined();
			}
		} finally {
			delete process.env.PRIVACY_WHITELIST_EMAILS;
		}
	});
});

describe('Coming-soon pages', () => {
	const at = (path: string) => settle(() => comingSoonLoad(event({ url: new URL(path, ORIGIN) })));

	it('redirects /overview and /pulse to /', async () => {
		expect(await at('/overview')).toEqual({ redirect: '/' });
		expect(await at('/pulse')).toEqual({ redirect: '/' });
	});

	it('names the unbuilt view a path is in', async () => {
		expect(await at('/narratives/N-0412')).toEqual({ value: { title: 'Narratives' } });
		expect(await at('/safety')).toEqual({ value: { title: 'High-harm' } });
	});

	it('is a 404 for anything else', async () => {
		expect(await at('/nope')).toMatchObject({ error: 404 });
	});
});

describe('Settings page', () => {
	const url = new URL('/settings', ORIGIN);

	it('loads the profile, password age and the clock', async () => {
		const { backend, locals } = await signedIn();
		const data = await settingsLoad(event({ locals, url }));
		assert(data);
		expect(data).toMatchObject({ profile, now: backend.clock.now() });
		expect(data.passwordChangedAt).toBeInstanceOf(Date);
	});

	describe('profile action', () => {
		it('saves, and the next load shows it', async () => {
			const { locals } = await signedIn();
			const result = await settingsActions.profile(
				event({
					locals,
					url,
					request: formRequest(`${url}?/profile`, { name: 'Aminata O.', email: profile.email })
				})
			);
			expect(result).toEqual({ form: 'profile', saved: true });
			const data = await settingsLoad(event({ locals, url }));
			assert(data);
			expect(data.profile.name).toBe('Aminata O.');
		});

		it('returns the values and the reason when it fails', async () => {
			const { locals } = await signedIn();
			const values = { name: 'Aminata', email: 'not-an-email' };
			const result = await settingsActions.profile(
				event({ locals, url, request: formRequest(`${url}?/profile`, values) })
			);
			expect(isActionFailure(result)).toBe(true);
			expect(result).toMatchObject({
				status: 400,
				data: { form: 'profile', values, error: 'Enter a valid email address.' }
			});
		});
	});

	describe('password action', () => {
		it('changes the password and swaps this session onto new tokens', async () => {
			const { backend, cookies, locals } = await signedIn();
			const oldAccess = cookies.get(ACCESS_COOKIE)!;
			const result = await settingsActions.password(
				event({
					cookies,
					locals,
					url,
					request: formRequest(`${url}?/password`, {
						currentPassword: DEMO_PASSWORD,
						newPassword: 'another-good-password',
						confirmPassword: 'another-good-password'
					})
				})
			);
			expect(result).toEqual({ form: 'password', updated: true });

			const newAccess = cookies.get(ACCESS_COOKIE)!;
			expect(newAccess).not.toBe(oldAccess);
			expect((await backend.services.auth.getSession(oldAccess)).ok).toBe(false);
			expect((await backend.services.auth.getSession(newAccess)).ok).toBe(true);
		});

		it('never sends passwords back', async () => {
			const { locals } = await signedIn();
			const result = await settingsActions.password(
				event({
					locals,
					url,
					request: formRequest(`${url}?/password`, {
						currentPassword: 'wrong-password-here',
						newPassword: 'another-good-password',
						confirmPassword: 'another-good-password'
					})
				})
			);
			expect(result).toMatchObject({
				status: 403,
				data: { form: 'password', error: 'Your current password is incorrect.' }
			});
			expect(JSON.stringify(result)).not.toContain('password-here');
			expect(JSON.stringify(result)).not.toContain('another-good-password');
		});
	});

	it('requests a password reset', async () => {
		const { locals } = await signedIn();
		const result = await settingsActions.resetPassword(event({ locals, url }));
		expect(result).toEqual({ form: 'password', resetSentTo: profile.email });
	});
});

describe('Login', () => {
	const login = (backend: TestBackend, path: string, fields: Record<string, string>) => {
		const cookies = createFakeCookies();
		const run = () =>
			loginActions.default(
				event({
					cookies,
					locals: testLocals(backend, null),
					request: formRequest(`${ORIGIN}${path}`, fields),
					url: new URL(path, ORIGIN)
				})
			);
		return settle(run).then((result) => ({ ...result, cookies }));
	};

	it('sets both token cookies and goes back where the person came from', async () => {
		const backend = createTestBackend();
		const result = await login(backend, '/login?redirectTo=%2Fsettings', {
			email: DEMO_EMAIL,
			password: DEMO_PASSWORD
		});
		expect(result.redirect).toBe('/settings');
		expect(result.cookies.get(ACCESS_COOKIE)).toBeDefined();
		expect(result.cookies.get(REFRESH_COOKIE)).toBeDefined();
	});

	it('never redirects off the site', async () => {
		const backend = createTestBackend();
		const result = await login(backend, '/login?redirectTo=%2F%2Fevil.example', {
			email: DEMO_EMAIL,
			password: DEMO_PASSWORD
		});
		expect(result.redirect).toBe('/');
	});

	it('keeps the email, not the password, when it fails', async () => {
		const backend = createTestBackend();
		const result = await login(backend, '/login', {
			email: DEMO_EMAIL,
			password: 'guessing-again'
		});
		expect(result.value).toMatchObject({
			status: 401,
			data: { email: DEMO_EMAIL, message: 'Email or password is incorrect.' }
		});
		expect(JSON.stringify(result.value)).not.toContain('guessing-again');
		expect(result.cookies.get(ACCESS_COOKIE)).toBeUndefined();
	});
});

describe('Signup', () => {
	const signup = (backend: TestBackend, fields: Record<string, string>) => {
		const cookies = createFakeCookies();
		return settle(() =>
			signupActions.default(
				event({
					cookies,
					locals: testLocals(backend, null),
					request: formRequest(`${ORIGIN}/signup`, fields)
				})
			)
		).then((result) => ({ ...result, cookies }));
	};

	const fields = {
		name: 'Kadi Sawadogo',
		email: 'k.sawadogo@wobsongo.org',
		password: 'kadi-password-1',
		confirmPassword: 'kadi-password-1'
	};

	it('creates the account, signs in and opens Overview', async () => {
		const result = await signup(createTestBackend(), fields);
		expect(result.redirect).toBe('/');
		expect(result.cookies.get(ACCESS_COOKIE)).toBeDefined();
	});

	it('keeps the name and email, not the passwords, when it fails', async () => {
		const result = await signup(createTestBackend(), { ...fields, email: DEMO_EMAIL });
		expect(result.value).toMatchObject({
			status: 409,
			data: {
				values: { name: fields.name, email: DEMO_EMAIL },
				message: 'An account with this email already exists.'
			}
		});
		expect(JSON.stringify(result.value)).not.toContain(fields.password);
	});
});

describe('Logout', () => {
	it('revokes the tokens, clears the cookies and goes to login', async () => {
		const { backend, cookies, locals } = await signedIn();
		const refreshToken = cookies.get(REFRESH_COOKIE)!;

		const result = await settle(() => logout(event({ cookies, locals })));
		expect(result).toEqual({ redirect: '/login' });
		expect(cookies.get(ACCESS_COOKIE)).toBeUndefined();
		expect(cookies.get(REFRESH_COOKIE)).toBeUndefined();
		expect((await backend.services.auth.refresh(refreshToken)).ok).toBe(false);
	});
});

describe('Language switch', () => {
	const choose = (fields: Record<string, string>, initial: Record<string, string> = {}) => {
		const cookies = createFakeCookies(initial);
		const request = formRequest(`${ORIGIN}/lang`, fields);
		return settle(() => setLanguage(event({ cookies, request }))).then((result) => ({
			...result,
			cookies
		}));
	};

	it('stores the language in a cookie the browser can read, and goes back', async () => {
		const result = await choose({ lang: 'en', redirectTo: '/settings?tab=bots' });
		expect(result.redirect).toBe('/settings?tab=bots');
		expect(result.cookies.get('wobsongo_lang')).toBe('en');
		expect(result.cookies.options.get('wobsongo_lang')).toMatchObject({
			path: '/',
			httpOnly: false,
			sameSite: 'lax'
		});
	});

	it('ignores unknown languages', async () => {
		const result = await choose({ lang: 'de', redirectTo: '/' }, { wobsongo_lang: 'fr' });
		expect(result.redirect).toBe('/');
		expect(result.cookies.get('wobsongo_lang')).toBe('fr');
	});

	it('never redirects off the site', async () => {
		const result = await choose({ lang: 'fr', redirectTo: '//evil.example' });
		expect(result.redirect).toBe('/');
	});
});
