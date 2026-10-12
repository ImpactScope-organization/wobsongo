/**
 * Display labels for domain values. English only for now; this is the one
 * place to swap for i18n messages later.
 */
import type { ReviewStatus, Verdict } from '#lib/domain/narrative.js';
import type { Platform } from '#lib/domain/platform.js';
import type { ReviewDecision } from '#lib/domain/review.js';

export const VERDICT_LABEL: Record<Verdict, string> = {
	false: 'False',
	misleading: 'Misleading',
	unverified: 'Unverified'
};

export const REVIEW_STATUS_LABEL: Record<ReviewStatus, string> = {
	'needs-check': 'Needs check',
	'auto-checked': 'Auto-checked',
	'expert-verified': 'Expert-verified',
	'not-in-kb': 'Not in KB'
};

export const PLATFORM_LABEL: Record<Platform, string> = {
	whatsapp: 'WhatsApp (bot)',
	facebook: 'Facebook',
	tiktok: 'TikTok',
	x: 'X',
	other: 'Other'
};

/** Shown after an expert decides on a review item. */
export const REVIEW_OUTCOME_LABEL: Record<ReviewDecision, string> = {
	approve: 'Approved',
	override: 'Override saved · reply updated',
	dismiss: 'Dismissed as irrelevant',
	'add-fact-check': 'Fact-check draft created'
};
