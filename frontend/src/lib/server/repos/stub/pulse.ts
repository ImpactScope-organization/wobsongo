import type { PulseSnapshot } from '#lib/domain/pulse.js';
import type { TimeRange } from '#lib/domain/time-range.js';
import { pulseSnapshot } from '#lib/fixtures/pulse.js';
import type { Clock } from '#lib/utils/clock.js';
import { ok } from '../../result.js';
import type { PulseRepo } from '../../services/pulse.js';
import { simulatedFailure } from './store.js';

/**
 * The fixture is the last 24 h; longer windows scale it so switching the range
 * visibly changes the page. Messages grow with the window, narratives less so.
 */
const SCALE: Record<TimeRange, { messages: number; narratives: number }> = {
	'24h': { messages: 1, narratives: 1 },
	'7d': { messages: 6.4, narratives: 1.6 },
	'30d': { messages: 25, narratives: 2.5 }
};

function scaled(snapshot: PulseSnapshot, range: TimeRange): PulseSnapshot {
	const { messages, narratives } = SCALE[range];
	if (messages === 1 && narratives === 1) return snapshot;

	const times = (value: number, factor: number) => Math.round(value * factor);
	return {
		...snapshot,
		messages: {
			current: times(snapshot.messages.current, messages),
			previous: times(snapshot.messages.previous, messages),
			trend: snapshot.messages.trend.map((value) => times(value, messages))
		},
		activeNarratives: {
			count: times(snapshot.activeNarratives.count, narratives),
			new: times(snapshot.activeNarratives.new, narratives)
		},
		rated: {
			false: times(snapshot.rated.false, narratives),
			misleading: times(snapshot.rated.misleading, narratives),
			unverified: times(snapshot.rated.unverified, narratives),
			accurate: times(snapshot.rated.accurate, narratives)
		},
		narratives: snapshot.narratives.map((narrative) => ({
			...narrative,
			volume: times(narrative.volume, messages)
		})),
		platformCounts: snapshot.platformCounts.map((count) => ({
			...count,
			count: times(count.count, messages)
		})),
		// An anomaly is an hours-long spike; it only shows in the 24 h view.
		anomaly: null
	};
}

export function createStubPulseRepo(clock: Clock): PulseRepo {
	return {
		async getSnapshot(scope, { range }) {
			const simulated = simulatedFailure(scope.accountId);
			if (simulated) return simulated;
			return ok({ ...scaled(structuredClone(pulseSnapshot), range), updatedAt: clock.now() });
		}
	};
}
