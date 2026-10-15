/**
 * Sample data for the Pulse view, taken from the Infodemic Dashboard design.
 * Shared by stories, component tests and the stub repositories, so
 * treat it as read-only.
 */
import type { Anomaly, HighHarmSummary, NarrativeSummary } from '#lib/domain/narrative.js';
import type { PlatformCount } from '#lib/domain/platform.js';
import type { PulseSnapshot } from '#lib/domain/pulse.js';
import type { ActivityEntry } from '#lib/domain/review.js';

/** "Now" for every Pulse fixture: Monday 12 Oct 2026, 14:05 GMT. */
export const PULSE_NOW = new Date('2026-10-12T14:05:00Z');

export const narratives: NarrativeSummary[] = [
	{
		id: 'N-0412',
		title: 'Compulsory contraceptive trial for schoolgirls',
		verdict: 'false',
		status: 'needs-check',
		domain: 'Family planning',
		harm: 4,
		volume: 1284,
		growthPct: 412,
		trend: [1, 1, 1, 2, 1, 2, 2, 3, 4, 7, 12, 20]
	},
	{
		id: 'N-0388',
		title: 'Implants cause permanent infertility',
		verdict: 'false',
		status: 'expert-verified',
		domain: 'Family planning',
		harm: 3,
		volume: 962,
		growthPct: 38,
		trend: [8, 9, 8, 10, 9, 11, 10, 12, 11, 12, 13, 13]
	},
	{
		id: 'N-0405',
		title: 'Potash drink ends an unwanted pregnancy',
		verdict: 'false',
		status: 'needs-check',
		domain: 'Safe abortion & self-care',
		harm: 5,
		volume: 211,
		growthPct: 96,
		trend: [2, 2, 3, 2, 3, 4, 3, 5, 5, 6, 7, 8]
	},
	{
		id: 'N-0361',
		title: 'HPV vaccine causes sterility in girls',
		verdict: 'misleading',
		status: 'auto-checked',
		domain: 'Adolescent SRH',
		harm: 3,
		volume: 640,
		growthPct: 12,
		trend: [9, 10, 9, 9, 10, 9, 10, 10, 9, 10, 11, 10]
	},
	{
		id: 'N-0409',
		title: 'Free clinic condoms are deliberately punctured',
		verdict: 'unverified',
		status: 'not-in-kb',
		domain: 'STIs / HIV',
		harm: 3,
		volume: 318,
		growthPct: 64,
		trend: [1, 1, 2, 2, 2, 3, 3, 4, 4, 5, 6, 7]
	}
];

export const anomaly: Anomaly = {
	narrativeId: 'N-0412',
	narrativeTitle: 'Compulsory contraceptive trial for schoolgirls',
	sigma: 3.4,
	growthPct: 412,
	windowHours: 3
};

export const platformCounts: PlatformCount[] = [
	{ platform: 'whatsapp', count: 2693 },
	{ platform: 'facebook', count: 1571 },
	{ platform: 'tiktok', count: 1099 },
	{ platform: 'x', count: 272 },
	{ platform: 'other', count: 153 }
];

export const highHarmSummary: HighHarmSummary = { awaitingReview: 2, fastTracked: 1 };

/** Messages received per hour over the last 24 h, oldest first. */
export const messagesTrend = [30, 31, 29, 32, 31, 33, 32, 34, 33, 38, 46, 52, 57];

/** Narratives rated in the window, by verdict (plus accurate, for the meter). */
export const ratedBreakdown = { false: 14, misleading: 7, unverified: 9, accurate: 7 };

/** Everything on the Pulse page for the last 24 h, as the repository returns it. */
export const pulseSnapshot: PulseSnapshot = {
	region: 'Burkina Faso',
	updatedAt: PULSE_NOW,
	messages: { current: 8412, previous: 5129, trend: messagesTrend },
	activeNarratives: { count: 37, new: 6 },
	rated: ratedBreakdown,
	medianDebunkMinutes: { current: 142, previous: 124 },
	narratives,
	anomaly,
	platformCounts,
	highHarm: highHarmSummary
};

export const activity: ActivityEntry[] = [
	{
		id: 'a1',
		at: new Date('2026-10-12T14:02:00Z'),
		actor: 'Dr. A.',
		text: 'Viewed alert “Potash + lemon”'
	},
	{
		id: 'a2',
		at: new Date('2026-10-12T13:47:00Z'),
		actor: null,
		text: 'Flagged 14 messages for emergency review'
	},
	{
		id: 'a3',
		at: new Date('2026-10-11T09:14:00Z'),
		actor: 'Dr. A.',
		text: 'Validated harm score 5/5 for “Potash drink”'
	},
	{
		id: 'a4',
		at: new Date('2026-10-09T08:30:00Z'),
		actor: 'K. S.',
		text: 'Published audio debunk “Potash drinks are dangerous” (Mooré)'
	}
];
