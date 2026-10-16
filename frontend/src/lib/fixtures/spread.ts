/**
 * Sample data for the spread view, taken from the Infodemic Dashboard design.
 * Read-only; shared by stories, tests and stubs.
 */
import type { MessageFormat, RegionCount } from '#lib/domain/narrative-detail.js';
import type { Platform } from '#lib/domain/platform.js';
import type {
	DailyIntents,
	LanguageFormatMix,
	NarrativeLag,
	WeeklyCounts
} from '#lib/domain/spread.js';
import { toCount, toTotal } from '#lib/domain/count.js';
import { narratives, OVERVIEW_NOW } from './overview.js';

export const SPREAD_NOW = OVERVIEW_NOW;

const DAY = 86_400_000;
/** Monday of the first of eight full weeks before SPREAD_NOW (W34–W41). */
const FIRST_WEEK = new Date('2026-08-17T00:00:00Z');
const week = (index: number) => new Date(FIRST_WEEK.getTime() + index * 7 * DAY);

export const platformWeeks: WeeklyCounts<Platform>[] = [
	[52, 24, 14, 6, 4],
	[50, 25, 15, 6, 4],
	[49, 25, 16, 6, 4],
	[48, 26, 17, 5, 4],
	[47, 26, 17, 6, 4],
	[47, 27, 18, 5, 3],
	[46, 27, 19, 5, 3],
	[46, 27, 19, 5, 3]
].map(([whatsapp, facebook, tiktok, x, other], index) => ({
	weekStart: week(index),
	counts: { whatsapp, facebook, tiktok, x, other }
}));

export const formatWeeks: WeeklyCounts<MessageFormat>[] = [
	[44, 20, 26, 10],
	[45, 20, 25, 10],
	[47, 21, 23, 9],
	[48, 21, 22, 9],
	[49, 22, 20, 9],
	[50, 22, 19, 9],
	[51, 23, 18, 8],
	[52, 23, 17, 8]
].map(([audio, video, text, image], index) => ({
	weekStart: week(index),
	counts: { audio, video, text, image }
}));

const lag = (index: number, origin: Platform, lagHours: number): NarrativeLag => ({
	narrativeId: narratives[index].id,
	title: narratives[index].title,
	origin,
	lagHours
});

export const lags: NarrativeLag[] = [
	lag(0, 'tiktok', 18),
	lag(1, 'facebook', 41),
	lag(2, 'tiktok', 9),
	lag(3, 'facebook', 63),
	lag(4, 'x', 30)
];

export const languageFormats: LanguageFormatMix[] = [
	['Mooré', 61, 18, 12, 9],
	['French', 22, 31, 34, 13],
	['Dioula', 58, 20, 14, 8],
	['Fulfulde', 70, 12, 10, 8],
	['Slang / Nouchi', 25, 40, 28, 7]
].map(([language, audio, video, text, image]) => ({
	language: language as string,
	counts: { audio, video, text, image } as Record<MessageFormat, number>
}));

/** Monday to Sunday, the last full week. */
export const dailyIntents: DailyIntents[] = [
	[58, 36, 6],
	[55, 38, 7],
	[61, 33, 6],
	[49, 35, 16],
	[52, 37, 11],
	[57, 36, 7],
	[60, 35, 5]
].map(([verification, general, outOfScope], index) => ({
	day: new Date(week(7).getTime() + index * DAY),
	counts: { verification, general, 'out-of-scope': outOfScope }
}));

export const allRegions: RegionCount[] = [
	{ region: 'Centre', count: toCount(3120), topTopic: 'Contraceptive trial rumor' },
	{ region: 'Hauts-Bassins', count: toCount(1240), topTopic: 'Implant infertility' },
	{ region: 'Centre-Nord', count: toCount(860), topTopic: 'Potash abortion drink' },
	{ region: 'Plateau-Central', count: toCount(610), topTopic: 'HPV vaccine sterility' },
	{ region: 'Boucle du Mouhoun', count: toCount(340), topTopic: 'Menstrual taboos' },
	{ region: 'Est', count: toCount(42), topTopic: 'HPV vaccine sterility' },
	{ region: 'Sahel', count: toCount(7) }
];

/** Respondents who shared a location with the bot, last 30 d (6,219, rounded: Sahel is suppressed). */
export const allRegionRespondents = toTotal(
	6219,
	allRegions.map((region) => region.count)
);
