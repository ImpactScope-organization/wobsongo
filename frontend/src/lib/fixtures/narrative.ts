/**
 * Sample data for the narrative detail view (narrative N-0412), taken from the
 * Infodemic Dashboard design. Read-only; shared by stories, tests and stubs.
 */
import type {
	DebunkFormat,
	EngineVerdict,
	FormatCount,
	LanguageCount,
	NarrativeDetail,
	NarrativeSpread,
	PlatformSeries,
	RegionCount,
	SampleMessage
} from '#lib/domain/narrative-detail.js';
import type { Platform } from '#lib/domain/platform.js';
import type { ActivityEntry } from '#lib/domain/review.js';
import { toCount, toTotal } from '#lib/domain/count.js';
import { narratives, PULSE_NOW } from './pulse.js';

export const NARRATIVE_NOW = PULSE_NOW;

export const narrative: NarrativeDetail = {
	...narratives[0],
	summary:
		'Claim that schoolgirls will receive contraceptive injections in a compulsory government trial, without parental consent.',
	volume7d: 2031,
	growthFactor6h: 4.1,
	harmValidated: true,
	priority: 87
};

export const engineVerdict: EngineVerdict = {
	verdict: 'false',
	confidence: 0.91,
	explanation:
		'Matched to “No compulsory contraception trials in Burkina Faso” (match 0.78). No such trial is registered with the Ministry of Health.'
};

const HOUR = 60 * 60 * 1000;
const WINDOW_HOURS = 72;
const windowStart = NARRATIVE_NOW.getTime() - WINDOW_HOURS * HOUR;

/**
 * Hourly mentions for one platform: zero before `startHour`, then a smooth
 * rise towards `peak` around `peakHour`, easing to `tail` afterwards.
 */
function series(
	platform: Platform,
	startHour: number,
	peakHour: number,
	peak: number,
	tail: number
): PlatformSeries {
	const points = Array.from({ length: WINDOW_HOURS + 1 }, (_, hour) => {
		let count = 0;
		if (hour >= startHour) {
			const rise = Math.min(1, (hour - startHour) / Math.max(1, peakHour - startHour));
			const fall = hour > peakHour ? Math.exp(-(hour - peakHour) / 18) : 1;
			count = Math.round(peak * Math.sin((rise * Math.PI) / 2) * fall + tail * (1 - fall));
		}
		return { at: new Date(windowStart + hour * HOUR), count };
	});
	return { platform, points };
}

export const spread: NarrativeSpread = {
	series: [
		series('tiktok', 2, 14, 30, 10),
		series('facebook', 8, 30, 26, 16),
		series('whatsapp', 20, 56, 62, 54)
	],
	firstSeen: { platform: 'tiktok', at: new Date(windowStart + 2 * HOUR + 10 * 60 * 1000) },
	firstOnWhatsApp: new Date(windowStart + 20 * HOUR + 25 * 60 * 1000)
};

export const reviewLog: ActivityEntry[] = [
	{
		id: 'r3',
		at: new Date('2026-10-12T13:52:00Z'),
		actor: null,
		text: 'API verdict “False” (0.91) attached automatically'
	},
	{
		id: 'r2',
		at: new Date('2026-10-12T13:40:00Z'),
		actor: 'Dr. A.',
		text: 'Harm 4/5 validated by Dr. A. (clinical advisor)'
	},
	{
		id: 'r1',
		at: new Date('2026-10-12T11:05:00Z'),
		actor: null,
		text: 'Narrative created: 3 clusters merged'
	}
];

export const languages: LanguageCount[] = [
	{ language: 'Mooré', count: 1178 },
	{ language: 'French', count: 488 },
	{ language: 'Dioula', count: 244 },
	{ language: 'Fulfulde', count: 81 },
	{ language: 'Other', count: 40 }
];

export const formats: FormatCount[] = [
	{ format: 'audio', count: 1056 },
	{ format: 'video', count: 467 },
	{ format: 'text', count: 345 },
	{ format: 'image', count: 163 }
];

export const regions: RegionCount[] = [
	{ region: 'Centre', count: toCount(412) },
	{ region: 'Centre-Nord', count: toCount(188) },
	{ region: 'Plateau-Central', count: toCount(140) },
	{ region: 'Hauts-Bassins', count: toCount(96) },
	{ region: 'Centre-Ouest', count: toCount(52) },
	{ region: 'Sahel', count: toCount(6) },
	{ region: 'Est', count: toCount(8) }
];

/** Respondents who shared a location with the bot (902, rounded: Sahel and Est are suppressed). */
export const regionRespondents = toTotal(
	902,
	regions.map((region) => region.count)
);

export const samples: SampleMessage[] = [
	{
		id: 's1',
		platform: 'whatsapp',
		formatLabel: 'Voice note 0:42',
		language: 'Mooré',
		text: 'They say at the school in [place] they will inject the girls next week without telling parents. Is it true?',
		note: 'Auto-transcribed, translated from Mooré'
	},
	{
		id: 's2',
		platform: 'tiktok',
		formatLabel: 'Video',
		language: 'Français',
		text: 'Le gouvernement va tester la contraception obligatoire sur nos filles. Partagez avant que ça soit supprimé.',
		note: 'Original caption · account anonymized'
	},
	{
		id: 's3',
		platform: 'facebook',
		formatLabel: 'Image + text',
		language: 'Français',
		text: '[Groupe] Liste des écoles concernées par l’essai. Gardez vos enfants à la maison lundi.',
		note: 'Public group · ~2.4k shares'
	}
];

export const debunkFormats: DebunkFormat[] = [
	{ id: 'd1', format: 'audio', language: 'Mooré', reachPct: 58, available: true },
	{ id: 'd2', format: 'text', language: 'French', reachPct: 24, available: true },
	{ id: 'd3', format: 'audio', language: 'Dioula', reachPct: 12, available: false },
	{ id: 'd4', format: 'audio', language: 'Fulfulde', reachPct: 4, available: false }
];
