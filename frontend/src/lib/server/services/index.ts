/**
 * The service layer. Pages and hooks use `Services`; which repositories sit
 * underneath (stub or backend) is decided once, in `backend.ts`.
 */
import { createAccountService, type AccountRepo, type AccountService } from './account.js';
import { createAuthService, type AuthRepo, type AuthService } from './auth.js';
import { createMonitorService, type MonitorRepo, type MonitorService } from './monitor.js';
import { createOverviewService, type OverviewRepo, type OverviewService } from './overview.js';

export interface Repos {
	auth: AuthRepo;
	account: AccountRepo;
	overview: OverviewRepo;
	monitor: MonitorRepo;
}

export interface Services {
	auth: AuthService;
	account: AccountService;
	overview: OverviewService;
	monitor: MonitorService;
}

export function createServices(repos: Repos): Services {
	const overview = createOverviewService(repos.overview);
	return {
		auth: createAuthService(repos.auth),
		account: createAccountService(repos.account),
		overview,
		monitor: createMonitorService(repos.monitor)
	};
}

export type { AccountService, AccountSettings, PasswordChange, ProfileChange } from './account.js';
export type { AuthService, Session, TokenPair } from './auth.js';
export type { MonitorService } from './monitor.js';
export type { OverviewService } from './overview.js';
