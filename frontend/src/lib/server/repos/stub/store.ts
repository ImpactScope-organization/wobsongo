/**
 * In-memory state behind the stub repositories, so a demo can sign up, sign
 * in and edit settings. One store lives as long as the server process; tests
 * create their own.
 */
import type { AccountProfile } from '#lib/domain/account.js';
import type { ProcessingRun } from '#lib/domain/monitor.js';
import { passwordChangedAt, profile } from '#lib/fixtures/settings.js';
import { failure, type Failure } from '../../result.js';

export interface StubAccount {
	id: string;
	profile: AccountProfile;
	password: string;
	passwordChangedAt: Date | null;
}

export interface StubToken {
	accountId: string;
	expiresAt: Date;
	/** For access tokens: the refresh token they were issued with, so logout revokes both. */
	refreshToken?: string;
}

export interface StubStore {
	accounts: Map<string, StubAccount>;
	accessTokens: Map<string, StubToken>;
	refreshTokens: Map<string, StubToken>;
	/** Manual Monitor runs started in this process, newest first. */
	monitorRuns: ProcessingRun[];
	/** Last number used for a new account id. */
	sequence: number;
}

/** The seeded reviewer: sign in with this email and `DEMO_PASSWORD`. */
export const DEMO_EMAIL = profile.email;
export const DEMO_PASSWORD = 'wobsongo-demo-2026';
export const DEMO_ACCOUNT_ID = 'acc-000';

export function createStubStore(): StubStore {
	const demo: StubAccount = {
		id: DEMO_ACCOUNT_ID,
		profile: structuredClone(profile),
		password: DEMO_PASSWORD,
		passwordChangedAt
	};
	return {
		accounts: new Map([[demo.id, demo]]),
		accessTokens: new Map(),
		refreshTokens: new Map(),
		monitorRuns: [],
		sequence: 0
	};
}

export function findAccountByEmail(store: StubStore, email: string): StubAccount | undefined {
	return [...store.accounts.values()].find((account) => account.profile.email === email);
}

/** Emails at this domain make the stub fail as if the backend were down. */
export const ERROR_EMAIL_DOMAIN = '@error.test';

/**
 * The stub conventions from tr.md: an id ending in "404" is not found, one
 * ending in "500" is a server error, and anything else behaves normally.
 */
export function simulatedFailure(id: string): Failure | null {
	if (id.endsWith('404')) return failure(404, 'Not found.');
	if (id.endsWith('500') || id.endsWith(ERROR_EMAIL_DOMAIN))
		return failure(500, 'Stub server error.');
	return null;
}
