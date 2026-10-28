import { describe, expect, it } from 'vitest';
import {
	botLinkMessage,
	initials,
	newPasswordError,
	normalizeEmail,
	passwordChangeError,
	profileError
} from './account';
import { toCount, toTotal } from './count';
import {
	costPerRun,
	hasPendingPosts,
	isFinalStatus,
	latestFinishedRun,
	selectRun,
	successRate,
	volumeTotals
} from './monitor';
import { isHighHarm, isSpiking } from './narrative';
import {
	growthPct,
	matchesTopic,
	misinfoShare,
	parseOverviewTopic,
	topCreators,
	topNarratives,
	trendDirection
} from './overview';
import { parseReviewFormDecision } from './review';
import { parseTimeRange } from './time-range';
import { whatsAppLagHours } from './narrative-detail';
import { fixedClock } from '../utils/clock';
import { narratives, sampleCreators } from '../fixtures/overview';
import { processingRuns, runningRun } from '../fixtures/monitor';

describe('toCount', () => {
	it('suppresses counts below the minimum cell size', () => {
		expect(toCount(6)).toEqual({ kind: 'suppressed', threshold: 10 });
		expect(toCount(4, 5)).toEqual({ kind: 'suppressed', threshold: 5 });
	});

	it('keeps counts at or above the minimum cell size', () => {
		expect(toCount(10)).toEqual({ kind: 'value', value: 10 });
		expect(toCount(412)).toEqual({ kind: 'value', value: 412 });
	});
});

describe('isHighHarm', () => {
	it('is true from harm 4 upwards', () => {
		expect(isHighHarm(3)).toBe(false);
		expect(isHighHarm(4)).toBe(true);
		expect(isHighHarm(5)).toBe(true);
	});
});

describe('isSpiking', () => {
	it('is true only above 50% growth', () => {
		expect(isSpiking(50)).toBe(false);
		expect(isSpiking(51)).toBe(true);
		expect(isSpiking(-80)).toBe(false);
	});
});

describe('parseTimeRange', () => {
	it('accepts known ranges', () => {
		expect(parseTimeRange('7d')).toBe('7d');
	});

	it('falls back to 24h for missing or unknown values', () => {
		expect(parseTimeRange(null)).toBe('24h');
		expect(parseTimeRange('1y')).toBe('24h');
	});
});

describe('fixedClock', () => {
	it('always returns the same instant, as a fresh Date', () => {
		const clock = fixedClock('2026-10-12T14:05:00Z');
		const first = clock.now();
		first.setFullYear(2000);
		expect(clock.now().toISOString()).toBe('2026-10-12T14:05:00.000Z');
	});
});

describe('whatsAppLagHours', () => {
	const firstSeen = { platform: 'tiktok' as const, at: new Date('2026-10-12T02:10:00Z') };

	it('is the rounded hours from first sighting to first WhatsApp submission', () => {
		expect(
			whatsAppLagHours({
				series: [],
				firstSeen,
				firstOnWhatsApp: new Date('2026-10-12T20:25:00Z')
			})
		).toBe(18);
	});

	it('is null before the narrative reaches WhatsApp', () => {
		expect(whatsAppLagHours({ series: [], firstSeen, firstOnWhatsApp: null })).toBeNull();
	});

	it('is never negative when WhatsApp saw it first', () => {
		expect(
			whatsAppLagHours({
				series: [],
				firstSeen,
				firstOnWhatsApp: new Date('2026-10-12T01:00:00Z')
			})
		).toBe(0);
	});
});

describe('parseReviewFormDecision', () => {
	it('accepts what a review form posts, including undo', () => {
		for (const value of ['approve', 'dismiss', 'add-fact-check', 'undo']) {
			expect(parseReviewFormDecision(value)).toBe(value);
		}
	});

	it('rejects override, which goes through the editor page, and anything else', () => {
		expect(parseReviewFormDecision('override')).toBeNull();
		expect(parseReviewFormDecision(null)).toBeNull();
		expect(parseReviewFormDecision('APPROVE')).toBeNull();
	});
});

describe('toTotal', () => {
	it('keeps the exact total when nothing is suppressed', () => {
		expect(toTotal(6219, [toCount(3120), toCount(42)])).toEqual({
			value: 6219,
			approximate: false
		});
	});

	it('rounds the total when a part is suppressed, so it cannot be subtracted back', () => {
		expect(toTotal(6219, [toCount(6212), toCount(7)])).toEqual({ value: 6220, approximate: true });
	});
});

describe('initials', () => {
	it('takes the first and last name, skipping titles', () => {
		expect(initials('Dr. Aminata Ouédraogo')).toBe('AO');
		expect(initials('Kader Sawadogo Zongo')).toBe('KZ');
		expect(initials('awa')).toBe('A');
		expect(initials('  ')).toBe('?');
	});
});

describe('botLinkMessage', () => {
	it('prefixes the code so the bot never fact-checks it', () => {
		expect(botLinkMessage({ code: '482913' })).toBe('WOBSONGO-AUTH 482-913');
	});
});

describe('parseOverviewTopic', () => {
	it('accepts known topics and falls back to misinformation', () => {
		expect(parseOverviewTopic('all')).toBe('all');
		expect(parseOverviewTopic('misinfo')).toBe('misinfo');
		expect(parseOverviewTopic('rumours')).toBe('misinfo');
		expect(parseOverviewTopic(null)).toBe('misinfo');
	});
});

describe('matchesTopic', () => {
	it('counts false and misleading as misinformation; all adds unverified', () => {
		expect(matchesTopic('false', 'misinfo')).toBe(true);
		expect(matchesTopic('misleading', 'misinfo')).toBe(true);
		expect(matchesTopic('unverified', 'misinfo')).toBe(false);
		expect(matchesTopic('unverified', 'all')).toBe(true);
	});
});

describe('topNarratives', () => {
	it('filters to the topic and ranks by volume', () => {
		const misinfo = topNarratives(narratives, 'misinfo');
		expect(misinfo.map((narrative) => narrative.id)).toEqual([
			'N-0412',
			'N-0388',
			'N-0361',
			'N-0405'
		]);
		expect(topNarratives(narratives, 'all')).toHaveLength(narratives.length);
	});

	it('leaves the input untouched', () => {
		const before = narratives.map((narrative) => narrative.id);
		topNarratives(narratives, 'all');
		expect(narratives.map((narrative) => narrative.id)).toEqual(before);
	});
});

describe('topCreators', () => {
	it('ranks creators by volume (posts/claims count) highest first', () => {
		const ranked = topCreators(sampleCreators, 'all');
		expect(ranked.map((creator) => creator.id)).toEqual([
			'c2', // 52 posts
			'c3', // 41 posts
			'c1', // 38 posts
			'c5', // 34 posts
			'c4' // 27 posts
		]);
	});

	it('leaves the input untouched', () => {
		const before = sampleCreators.map((creator) => creator.id);
		topCreators(sampleCreators, 'all');
		expect(sampleCreators.map((creator) => creator.id)).toEqual(before);
	});
});

describe('growthPct', () => {
	it('is the change from the previous window in percent', () => {
		expect(growthPct({ current: 150, previous: 100 })).toBe(50);
		expect(growthPct({ current: 75, previous: 100 })).toBe(-25);
	});

	it('is null without a previous value', () => {
		expect(growthPct({ current: 10, previous: 0 })).toBeNull();
	});
});

describe('misinfoShare', () => {
	it('counts false and misleading out of every rated narrative', () => {
		expect(misinfoShare({ false: 14, misleading: 7, unverified: 9, accurate: 7 })).toEqual({
			misinfo: 21,
			total: 37
		});
	});
});

describe('trendDirection', () => {
	it('compares the last value with the first', () => {
		expect(trendDirection([1, 5, 3])).toBe('rising');
		expect(trendDirection([5, 9, 2])).toBe('falling');
		expect(trendDirection([4, 1, 4])).toBe('steady');
		expect(trendDirection([4])).toBe('steady');
	});
});

describe('normalizeEmail', () => {
	it('trims and lowercases', () => {
		expect(normalizeEmail('  A.Ouedraogo@WobSongo.org ')).toBe('a.ouedraogo@wobsongo.org');
	});
});

describe('profileError', () => {
	it('accepts a name and a valid email', () => {
		expect(profileError({ name: 'Aminata', email: 'a@b.org' })).toBeNull();
	});

	it('requires a name of at most 100 characters', () => {
		expect(profileError({ name: '  ', email: 'a@b.org' })).toBe('Enter your name.');
		expect(profileError({ name: 'x'.repeat(101), email: 'a@b.org' })).toMatch(/at most 100/);
	});

	it('requires a valid email', () => {
		expect(profileError({ name: 'Aminata', email: 'aminata' })).toBe(
			'Enter a valid email address.'
		);
		expect(profileError({ name: 'Aminata', email: 'a b@c.org' })).toBe(
			'Enter a valid email address.'
		);
	});
});

describe('newPasswordError', () => {
	it('requires 12 characters and a matching confirmation', () => {
		expect(newPasswordError({ newPassword: 'short', confirmPassword: 'short' })).toMatch(
			/at least 12/
		);
		expect(
			newPasswordError({ newPassword: 'long-enough-pass', confirmPassword: 'long-enough-pas' })
		).toBe('The passwords don’t match.');
		expect(
			newPasswordError({ newPassword: 'long-enough-pass', confirmPassword: 'long-enough-pass' })
		).toBeNull();
	});
});

describe('passwordChangeError', () => {
	const change = {
		currentPassword: 'old-password-123',
		newPassword: 'new-password-456',
		confirmPassword: 'new-password-456'
	};

	it('accepts a valid change', () => {
		expect(passwordChangeError(change)).toBeNull();
	});

	it('requires the current password', () => {
		expect(passwordChangeError({ ...change, currentPassword: '' })).toBe(
			'Enter your current password.'
		);
	});

	it('rejects reusing the current password', () => {
		expect(
			passwordChangeError({
				...change,
				newPassword: change.currentPassword,
				confirmPassword: change.currentPassword
			})
		).toMatch(/different/);
	});
});

describe('monitor', () => {
	const runs = [runningRun, ...processingRuns];

	it('selects the run by id, else the newest', () => {
		expect(selectRun(runs, processingRuns[2].id)).toBe(processingRuns[2]);
		expect(selectRun(runs, 'run-missing')).toBe(runningRun);
		expect(selectRun(runs, null)).toBe(runningRun);
		expect(selectRun([], null)).toBeUndefined();
	});

	it('finds the newest finished run', () => {
		expect(latestFinishedRun(runs)).toBe(processingRuns[0]);
		expect(latestFinishedRun([runningRun])).toBeUndefined();
	});

	it('knows which statuses are final', () => {
		expect(isFinalStatus('clustered')).toBe(true);
		expect(isFinalStatus('failed')).toBe(true);
		expect(isFinalStatus('transcribed')).toBe(false);
	});

	it('spots runs whose posts are still being processed', () => {
		expect(hasPendingPosts(processingRuns[0].result!)).toBe(true);
		expect(hasPendingPosts(processingRuns[1].result!)).toBe(false);
	});

	it('keeps every fixture run consistent', () => {
		for (const { result } of processingRuns) {
			const sum = (counts: Record<string, number>) =>
				Object.values(counts).reduce((total, count) => total + count, 0);
			expect(sum(result!.statusCounts)).toBe(result!.newPosts);
			expect(sum(result!.platformCounts)).toBe(result!.newPosts);
			expect(result!.statusCounts.failed).toBe(result!.failed);
		}
	});

	it('totals the volume and rates a bucket', () => {
		const start = new Date(0);
		expect(
			volumeTotals([
				{ start, processed: 90, failed: 10 },
				{ start, processed: 5, failed: 0 }
			])
		).toEqual({ processed: 95, failed: 10 });
		expect(successRate({ start, processed: 90, failed: 10 })).toBe(90);
		expect(successRate({ start, processed: 0, failed: 0 })).toBeNull();
	});

	it('averages the cost per run', () => {
		expect(costPerRun({ costUsd: 24, runCount: 12 })).toBe(2);
		expect(costPerRun({ costUsd: 0, runCount: 0 })).toBe(0);
	});
});
