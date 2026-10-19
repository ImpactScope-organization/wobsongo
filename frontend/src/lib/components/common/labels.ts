/**
 * Display labels for domain values. English only for now; this is the one
 * place to swap for i18n messages later.
 */
import type { AccountRole, BotPlatform } from '#lib/domain/account.js';
import type { HarmRisk } from '#lib/domain/high-harm.js';
import type { CheckReason, PriorityQuadrant } from '#lib/domain/operations.js';
import type { DismissReason, MessageFormat } from '#lib/domain/narrative-detail.js';
import type { RatingOutcome, ReviewStatus, Verdict } from '#lib/domain/narrative.js';
import type { Platform } from '#lib/domain/platform.js';
import type { OverviewTopic } from '#lib/domain/overview.js';
import type { ReviewDecision } from '#lib/domain/review.js';
import type { MessageIntent } from '#lib/domain/spread.js';
import type { TimeRange } from '#lib/domain/time-range.js';

export const VERDICT_LABEL: Record<Verdict, string> = {
	false: 'False',
	misleading: 'Misleading',
	unverified: 'Unverified'
};

export const RATING_OUTCOME_LABEL: Record<RatingOutcome, string> = {
	...VERDICT_LABEL,
	accurate: 'Accurate'
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

export const MESSAGE_FORMAT_LABEL: Record<MessageFormat, string> = {
	audio: 'Voice note',
	video: 'Video',
	text: 'Text',
	image: 'Image'
};

/** Short name of a platform, without qualifiers like "(bot)". */
export const PLATFORM_SHORT_LABEL: Record<Platform, string> = {
	whatsapp: 'WhatsApp',
	facebook: 'Facebook',
	tiktok: 'TikTok',
	x: 'X',
	other: 'Other'
};

export const DISMISS_REASON_LABEL: Record<DismissReason, { label: string; description: string }> = {
	'not-srh': { label: 'Not SRH-related', description: 'Off-topic for this platform' },
	'not-a-claim': { label: 'Not a claim', description: 'Greeting, opinion or spam' },
	duplicate: { label: 'Duplicate', description: 'Already covered by another narrative' },
	satire: { label: 'Satire or joke', description: 'Not meant as fact' }
};

export const HARM_RISK_LABEL: Record<HarmRisk, string> = {
	critical: 'Critical',
	high: 'High'
};

export const MESSAGE_INTENT_LABEL: Record<MessageIntent, { label: string; description: string }> = {
	verification: {
		label: 'Verification request',
		description: 'Forwarded message or link to check'
	},
	general: {
		label: 'General SRH question',
		description: 'Information request, e.g. how a method works'
	},
	'out-of-scope': {
		label: 'Out of scope',
		description:
			'Personal medical questions the platform does not handle. Tracked to improve bot replies'
	}
};

export const CHECK_REASON_LABEL: Record<CheckReason, string> = {
	spiking: 'Spiking',
	'low-confidence': 'Low confidence',
	'partly-true': 'Partly true'
};

export const PRIORITY_QUADRANT_LABEL: Record<PriorityQuadrant, string> = {
	critical: 'Critical priority',
	urgent: 'Urgent debunk',
	monitor: 'Monitor',
	campaign: 'Public campaign'
};

export const ACCOUNT_ROLE_LABEL: Record<AccountRole, string> = {
	user: 'Member',
	admin: 'Administrator',
	superadmin: 'Super administrator'
};

export const BOT_LABEL: Record<BotPlatform, string> = {
	whatsapp: 'WhatsApp',
	telegram: 'Telegram'
};

/** Two-letter tiles for the bots. */
export const BOT_MONOGRAM: Record<BotPlatform, string> = {
	whatsapp: 'WA',
	telegram: 'TG'
};

/** Window lengths, e.g. "Messages received, 24 h". */
export const TIME_RANGE_LABEL: Record<TimeRange, string> = {
	'24h': '24 h',
	'7d': '7 d',
	'30d': '30 d'
};

export const OVERVIEW_TOPIC_LABEL: Record<OverviewTopic, string> = {
	misinfo: 'Misinformation',
	all: 'All topics'
};
