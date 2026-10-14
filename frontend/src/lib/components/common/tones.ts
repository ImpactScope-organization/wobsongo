/**
 * Colour classes for categorical data, shared so a platform or format keeps
 * the same colour across every chart and legend.
 */
import type { BotPlatform } from '#lib/domain/account.js';
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
