/**
 * Display labels for domain values, in the current language. Each entry is a
 * getter that calls its message, so `VERDICT_LABEL.false` reads the language
 * of the request (or page) at the time it's read. Read them while rendering,
 * never into a module-level constant, or the first language sticks.
 */
import type { AccountRole, BotPlatform } from '#lib/domain/account.js';
import type { HarmRisk } from '#lib/domain/high-harm.js';
import type { CrawledPlatform, ProcessingStatus, RunType } from '#lib/domain/monitor.js';
import type { CheckReason, PriorityQuadrant } from '#lib/domain/operations.js';
import type { DismissReason, MessageFormat } from '#lib/domain/narrative-detail.js';
import type { RatingOutcome, ReviewStatus, Verdict } from '#lib/domain/narrative.js';
import type { Platform } from '#lib/domain/platform.js';
import type { OverviewTopic } from '#lib/domain/overview.js';
import type { ReviewDecision } from '#lib/domain/review.js';
import type { MessageIntent } from '#lib/domain/spread.js';
import type { TimeRange } from '#lib/domain/time-range.js';
import { m } from '#lib/paraglide/messages.js';

type Message = () => string;
type Messages = { readonly [key: string]: Message | Messages };
type Localized<T> = { readonly [K in keyof T]: T[K] extends Message ? string : Localized<T[K]> };

/** An object of the same shape whose leaves are read from `messages` on access. */
function localized<T extends Messages>(messages: T): Localized<T> {
	const result = {};
	for (const [key, value] of Object.entries(messages)) {
		const nested = typeof value === 'function' ? undefined : localized(value);
		Object.defineProperty(result, key, {
			enumerable: true,
			get: nested ? () => nested : () => (value as Message)()
		});
	}
	return result as Localized<T>;
}

const verdictMessages = {
	false: m.label_verdict_false,
	misleading: m.label_verdict_misleading,
	unverified: m.label_verdict_unverified
} satisfies Record<Verdict, Message>;

export const VERDICT_LABEL: Record<Verdict, string> = localized(verdictMessages);

export const RATING_OUTCOME_LABEL: Record<RatingOutcome, string> = localized({
	...verdictMessages,
	accurate: m.label_verdict_accurate
});

export const REVIEW_STATUS_LABEL: Record<ReviewStatus, string> = localized({
	'needs-check': m.label_review_status_needs_check,
	'auto-checked': m.label_review_status_auto_checked,
	'expert-verified': m.label_review_status_expert_verified,
	'not-in-kb': m.label_review_status_not_in_kb
});

export const PLATFORM_LABEL: Record<Platform, string> = localized({
	whatsapp: m.label_platform_whatsapp_bot,
	facebook: () => 'Facebook',
	tiktok: () => 'TikTok',
	x: () => 'X',
	other: m.label_platform_other
});

/** Shown after an expert decides on a review item. */
export const REVIEW_OUTCOME_LABEL: Record<ReviewDecision, string> = localized({
	approve: m.label_review_outcome_approve,
	override: m.label_review_outcome_override,
	dismiss: m.label_review_outcome_dismiss,
	'add-fact-check': m.label_review_outcome_add_fact_check
});

export const MESSAGE_FORMAT_LABEL: Record<MessageFormat, string> = localized({
	audio: m.label_format_audio,
	video: m.label_format_video,
	text: m.label_format_text,
	image: m.label_format_image
});

/** Short name of a platform, without qualifiers like "(bot)". */
export const PLATFORM_SHORT_LABEL: Record<Platform, string> = localized({
	whatsapp: () => 'WhatsApp',
	facebook: () => 'Facebook',
	tiktok: () => 'TikTok',
	x: () => 'X',
	other: m.label_platform_other
});

export const DISMISS_REASON_LABEL: Record<DismissReason, { label: string; description: string }> =
	localized({
		'not-srh': {
			label: m.label_dismiss_not_srh,
			description: m.label_dismiss_not_srh_description
		},
		'not-a-claim': {
			label: m.label_dismiss_not_a_claim,
			description: m.label_dismiss_not_a_claim_description
		},
		duplicate: {
			label: m.label_dismiss_duplicate,
			description: m.label_dismiss_duplicate_description
		},
		satire: { label: m.label_dismiss_satire, description: m.label_dismiss_satire_description }
	});

export const HARM_RISK_LABEL: Record<HarmRisk, string> = localized({
	critical: m.label_harm_risk_critical,
	high: m.label_harm_risk_high
});

export const MESSAGE_INTENT_LABEL: Record<MessageIntent, { label: string; description: string }> =
	localized({
		verification: {
			label: m.label_intent_verification,
			description: m.label_intent_verification_description
		},
		general: {
			label: m.label_intent_general,
			description: m.label_intent_general_description
		},
		'out-of-scope': {
			label: m.label_intent_out_of_scope,
			description: m.label_intent_out_of_scope_description
		}
	});

export const CHECK_REASON_LABEL: Record<CheckReason, string> = localized({
	spiking: m.label_check_reason_spiking,
	'low-confidence': m.label_check_reason_low_confidence,
	'partly-true': m.label_check_reason_partly_true
});

export const PRIORITY_QUADRANT_LABEL: Record<PriorityQuadrant, string> = localized({
	critical: m.label_quadrant_critical,
	urgent: m.label_quadrant_urgent,
	monitor: m.label_quadrant_monitor,
	campaign: m.label_quadrant_campaign
});

export const ACCOUNT_ROLE_LABEL: Record<AccountRole, string> = localized({
	user: m.label_role_user,
	admin: m.label_role_admin,
	superadmin: m.label_role_superadmin
});

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
export const TIME_RANGE_LABEL: Record<TimeRange, string> = localized({
	'24h': m.label_time_range_24h,
	'7d': m.label_time_range_7d,
	'30d': m.label_time_range_30d
});

export const OVERVIEW_TOPIC_LABEL: Record<OverviewTopic, string> = localized({
	misinfo: m.label_topic_misinfo,
	all: m.label_topic_all
});

export const RUN_TYPE_LABEL: Record<RunType, string> = localized({
	automatic: m.monitor_run_type_automatic,
	manual: m.monitor_run_type_manual
});

export const CRAWLED_PLATFORM_LABEL: Record<CrawledPlatform, string> = {
	facebook: 'Facebook',
	tiktok: 'TikTok',
	x: 'X',
	instagram: 'Instagram',
	youtube: 'YouTube'
};

export const PROCESSING_STATUS_LABEL: Record<ProcessingStatus, string> = localized({
	discovered: m.monitor_status_discovered,
	screened_out: m.monitor_status_screened_out,
	screened_in: m.monitor_status_screened_in,
	transcribed: m.monitor_status_transcribed,
	out_of_scope: m.monitor_status_out_of_scope,
	analyzed: m.monitor_status_analyzed,
	verified: m.monitor_status_verified,
	clustered: m.monitor_status_clustered,
	failed: m.monitor_status_failed
});

/** What each status means, e.g. "Out of SRH scope". */
export const PROCESSING_STATUS_DESCRIPTION: Record<ProcessingStatus, string> = localized({
	discovered: m.monitor_status_discovered_description,
	screened_out: m.monitor_status_screened_out_description,
	screened_in: m.monitor_status_screened_in_description,
	transcribed: m.monitor_status_transcribed_description,
	out_of_scope: m.monitor_status_out_of_scope_description,
	analyzed: m.monitor_status_analyzed_description,
	verified: m.monitor_status_verified_description,
	clustered: m.monitor_status_clustered_description,
	failed: m.monitor_status_failed_description
});
