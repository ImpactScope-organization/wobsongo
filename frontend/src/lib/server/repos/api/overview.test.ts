import { describe, expect, it } from 'vitest';
import type { EngagementWindow } from '#lib/domain/overview.js';
import { createApiRepos } from './index.js';
import { fixedClock } from '#lib/utils/clock.js';

const NOW = new Date('2026-11-05T12:00:00Z');
const clock = fixedClock(NOW);

const narrative = (over: Record<string, unknown>) => ({
	id: 'n1',
	statement: 'Potash drink ends a pregnancy',
	topic_name: 'Safe abortion & self-care',
	verdict: 'contradicted',
	harm: 5,
	status: 'needs_check',
	is_active: true,
	created_at: NOW.toISOString(),
	...over
});

function repos(items: unknown[], total = items.length, activity = { users: 0, messages: 0 }) {
	const sent: string[] = [];
	const fetch = async (input: RequestInfo | URL) => {
		const url = new URL((input as Request).url);
		sent.push(`${url.pathname}${url.search}`);
		if (url.pathname.endsWith('/conversations/stats')) {
			return Response.json({ request_id: 'r', status: 200, data: activity });
		}
		return Response.json({
			request_id: 'r',
			status: 200,
			data: { page: 1, per_page: 100, total_items: total, items }
		});
	};
	return {
		sent,
		repos: createApiRepos({
			baseUrl: 'https://api.example.org',
			fetch: fetch as typeof globalThis.fetch,
			clock
		})
	};
}

describe('API overview repo', () => {
	const snapshotOf = async (
		engagement: EngagementWindow,
		items: unknown[] = [],
		activity = { users: 0, messages: 0 }
	) => {
		const { sent, repos: r } = repos(items, items.length, activity);
		const result = await r.overview.getSnapshot({ accountId: 'a' }, { range: '24h', engagement });
		return { sent, result };
	};

	it('maps active narratives and derives the counts', async () => {
		const { sent, repos: r } = repos(
			[
				narrative({}),
				narrative({
					id: 'n2',
					verdict: 'mixed',
					harm: 2,
					status: 'expert_verified',
					created_at: '2026-09-01T00:00:00Z'
				}),
				narrative({ id: 'n3', verdict: 'supported', harm: 1 })
			],
			37
		);
		const result = await r.overview.getSnapshot(
			{ accountId: 'a' },
			{ range: '24h', engagement: 'all' }
		);
		expect(sent.some((request) => request.includes('is_active=true'))).toBe(true);
		if (!result.ok) throw new Error('expected ok');
		const s = result.data;
		expect(s.narratives.map((n) => [n.id, n.verdict, n.status])).toEqual([
			['n1', 'false', 'needs-check'],
			['n2', 'misleading', 'expert-verified']
		]);
		expect(s.rated).toEqual({ false: 1, misleading: 1, unverified: 0, accurate: 1 });
		expect(s.activeNarratives).toEqual({ count: 37, new: 2 });
		expect(s.highHarm.awaitingReview).toBe(1);
		expect(s.creators).toEqual([]);
	});

	it('reads the users and messages of the chosen window from the backend', async () => {
		const { result } = await snapshotOf('7d', [], { users: 3, messages: 41 });
		if (!result.ok) throw new Error('expected ok');
		expect(result.data.userQuestions).toEqual({ users: 3, questions: 41 });
	});

	it.each([
		['7d', '?days=7'],
		['15d', '?days=15'],
		['30d', '?days=30']
	] as const)('asks the backend for the last days of %s', async (engagement, query) => {
		const { sent } = await snapshotOf(engagement);
		expect(sent).toContain(`/api/v1/conversations/stats${query}`);
	});

	it('asks for all time when no days are given', async () => {
		const { sent } = await snapshotOf('all');
		expect(sent).toContain('/api/v1/conversations/stats');
	});

	it('keeps the status of a failed call', async () => {
		const r = createApiRepos({
			baseUrl: 'https://api.example.org',
			fetch: (async () =>
				Response.json(
					{ request_id: 'r', status: 401, error: 'unauthorized' },
					{ status: 401 }
				)) as typeof globalThis.fetch,
			clock
		});
		expect(
			await r.overview.getSnapshot({ accountId: 'a' }, { range: '7d', engagement: 'all' })
		).toMatchObject({ ok: false, status: 401 });
	});

	describe('when only one of the two backend calls fails', () => {
		const failingRepos = (failing: 'stats' | 'narratives', status: number) =>
			createApiRepos({
				baseUrl: 'https://api.example.org',
				fetch: (async (input: RequestInfo | URL) => {
					const url = new URL((input as Request).url);
					const isStats = url.pathname.endsWith('/conversations/stats');

					if (isStats === (failing === 'stats')) {
						return Response.json({ request_id: 'r', status, error: 'boom' }, { status });
					}
					if (isStats) {
						return Response.json({
							request_id: 'r',
							status: 200,
							data: { users: 3, messages: 41 }
						});
					}
					return Response.json({
						request_id: 'r',
						status: 200,
						data: { page: 1, per_page: 100, total_items: 0, items: [] }
					});
				}) as typeof globalThis.fetch,
				clock
			});

		it('fails when the user activity cannot be read', async () => {
			const r = failingRepos('stats', 403);
			expect(
				await r.overview.getSnapshot({ accountId: 'a' }, { range: '7d', engagement: '7d' })
			).toMatchObject({ ok: false, status: 403 });
		});

		it('fails when the narratives cannot be read', async () => {
			const r = failingRepos('narratives', 500);
			expect(
				await r.overview.getSnapshot({ accountId: 'a' }, { range: '7d', engagement: '7d' })
			).toMatchObject({ ok: false, status: 500 });
		});

		it('treats an unreachable backend as a 503', async () => {
			const r = createApiRepos({
				baseUrl: 'https://api.example.org',
				fetch: (async () => {
					throw new TypeError('network down');
				}) as typeof globalThis.fetch,
				clock
			});
			expect(
				await r.overview.getSnapshot({ accountId: 'a' }, { range: '7d', engagement: 'all' })
			).toMatchObject({ ok: false, status: 503 });
		});
	});

	it('takes "now" from the clock', async () => {
		const { result } = await snapshotOf('all', [
			narrative({ id: 'n1', created_at: '2026-11-05T11:00:00Z' }), // 1 h ago: new
			narrative({ id: 'n2', created_at: '2026-11-04T11:00:00Z' }) // 25 h ago: not new
		]);
		if (!result.ok) throw new Error('expected ok');
		expect(result.data.updatedAt).toEqual(NOW);
		expect(result.data.activeNarratives.new).toBe(1);
	});
});
