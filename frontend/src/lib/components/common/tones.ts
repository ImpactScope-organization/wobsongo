/**
 * Colour classes for categorical data, shared so a platform or format keeps
 * the same colour across every chart and legend.
 */
import type { BotPlatform } from '#lib/domain/account.js';
import type { CrawledPlatform, ProcessingStatus } from '#lib/domain/monitor.js';
import type { MessageFormat } from '#lib/domain/narrative-detail.js';
import type { Platform } from '#lib/domain/platform.js';

export interface Tone {
	/** Fill with readable text on top, for bars that print values. */
	bar: string;
	/** Fill only, for legend swatches and thin bars. */
	swatch: string;
}

export const PLATFORM_TONE: Record<Platform, Tone> = {
	whatsapp: { bar: 'bg-platform-whatsapp text-white', swatch: 'bg-platform-whatsapp' },
	facebook: { bar: 'bg-platform-facebook text-white', swatch: 'bg-platform-facebook' },
	tiktok: { bar: 'bg-platform-tiktok text-white', swatch: 'bg-platform-tiktok' },
	x: { bar: 'bg-platform-x text-white', swatch: 'bg-platform-x' },
	other: { bar: 'bg-platform-other text-foreground', swatch: 'bg-platform-other' }
};

export const FORMAT_TONE: Record<MessageFormat, Tone> = {
	audio: { bar: 'bg-primary text-primary-foreground', swatch: 'bg-primary' },
	video: { bar: 'bg-muted-foreground text-white', swatch: 'bg-muted-foreground' },
	text: {
		bar: 'bg-verdict-unverified-strong text-foreground',
		swatch: 'bg-verdict-unverified-strong'
	},
	image: { bar: 'bg-platform-other text-foreground', swatch: 'bg-platform-other' }
};

/** Monogram tiles for the messaging bots. */
export const BOT_TONE: Record<BotPlatform, string> = {
	whatsapp: 'bg-status-verified-foreground text-white',
	telegram: 'bg-platform-facebook text-white'
};

/** Crawled platforms; the ones shared with `Platform` keep its colours. */
export const CRAWLED_PLATFORM_TONE: Record<CrawledPlatform, Tone> = {
	facebook: PLATFORM_TONE.facebook,
	tiktok: PLATFORM_TONE.tiktok,
	x: PLATFORM_TONE.x,
	instagram: { bar: 'bg-platform-instagram text-white', swatch: 'bg-platform-instagram' },
	youtube: { bar: 'bg-platform-youtube text-white', swatch: 'bg-platform-youtube' }
};

/** Pipeline stages: greys for screening, blue to green as claims get checked, signal for failures. */
export const PROCESSING_STATUS_TONE: Record<ProcessingStatus, string> = {
	discovered: 'bg-platform-other',
	screened_out: 'bg-faint-foreground',
	screened_in: 'bg-verdict-unverified-strong',
	transcribed: 'bg-muted-foreground',
	out_of_scope: 'bg-pipeline-out-of-scope',
	analyzed: 'bg-platform-facebook',
	verified: 'bg-pipeline-verified',
	clustered: 'bg-verdict-accurate-strong',
	failed: 'bg-signal'
};
