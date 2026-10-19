/**
 * The service layer. Pages and hooks use `Services`; which repositories sit
 * underneath (stub or backend) is decided once, in `backend.ts`.
 */
import { createAccountService, type AccountRepo, type AccountService } from './account.js';
import { createAuthService, type AuthRepo, type AuthService } from './auth.js';
import { createOverviewService, type OverviewRepo, type OverviewService } from './overview.js';

export interface Repos {
	auth: AuthRepo;
	account: AccountRepo;
	overview: OverviewRepo;
}

export interface Services {
	auth: AuthService;
	account: AccountService;
	overview: OverviewService;
}

export function createServices(repos: Repos): Services {
	const overview = createOverviewService(repos.overview);
	return {
		auth: createAuthService(repos.auth),
		account: createAccountService(repos.account),
		overview
	};
}

export type { AccountService, AccountSettings, PasswordChange, ProfileChange } from './account.js';
export type { AuthService, Session, TokenPair } from './auth.js';
export type { OverviewService } from './overview.js';
