/**
 * The service layer. Pages and hooks use `Services`; which repositories sit
 * underneath (stub or backend) is decided once, in `backend.ts`.
 */
import { createAccountService, type AccountRepo, type AccountService } from './account.js';
import { createAuthService, type AuthRepo, type AuthService } from './auth.js';
import { createPulseService, type PulseRepo, type PulseService } from './pulse.js';

export interface Repos {
	auth: AuthRepo;
	account: AccountRepo;
	pulse: PulseRepo;
}

export interface Services {
	auth: AuthService;
	account: AccountService;
	pulse: PulseService;
}

export function createServices(repos: Repos): Services {
	return {
		auth: createAuthService(repos.auth),
		account: createAccountService(repos.account),
		pulse: createPulseService(repos.pulse)
	};
}

export type { AccountService, AccountSettings, PasswordChange, ProfileChange } from './account.js';
export type { AuthService, Session, TokenPair } from './auth.js';
export type { PulseService } from './pulse.js';
