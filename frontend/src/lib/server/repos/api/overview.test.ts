import { describe, expect, it } from 'vitest';
import type { EngagementWindow } from '#lib/domain/overview.js';
import { PLATFORMS } from '#lib/domain/platform.js';
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

interface Backend {
	items?: unknown[];
	total?: number;
	activity?: { users: number; messages: number };
	volume?: unknown;
	platforms?: unknown;
	// Answers a request itself; return nothing to fall through to the defaults.
	respond?: (url: URL) => Response | undefined;
}

const failed = (status: number) =>
	Response.json({ request_id: 'r', status, error: 'boom' }, { status });

// Answers every endpoint the Overview reads, so a test only overrides what it checks.
function backendFetch(backend: Backend, sent: string[]): typeof globalThis.fetch {
	const {
		items = [],
		total = items.length,
		activity = { users: 0, messages: 0 },
		volume = { current: 0, previous: 0, trend: [] },
		platforms = [],
		respond
	} = backend;

	return (async (input: RequestInfo | URL) => {
		const url = new URL((input as Request).url);
		sent.push(`${url.pathname}${url.search}`);

		const custom = respond?.(url);
		if (custom) return custom;

		if (url.pathname.endsWith('/conversations/stats')) {
			return Response.json({ request_id: 'r', status: 200, data: activity });
		}
		if (url.pathname.endsWith('/conversations/volume')) {
			return Response.json({ request_id: 'r', status: 200, data: volume });
		}
		if (url.pathname.endsWith('/claims/platforms')) {
			return Response.json({ request_id: 'r', status: 200, data: platforms });
		}
		const page = Number(url.searchParams.get('page') ?? 1);
		const perPage = Number(url.searchParams.get('per_page') ?? 20);
		return Response.json({
			request_id: 'r',
			status: 200,
			data: {
				page,
				per_page: perPage,
				total_items: total,
				items: items.slice((page - 1) * perPage, page * perPage)
			}
		});
	}) as typeof globalThis.fetch;
}

function reposFor(backend: Backend) {
	const sent: string[] = [];
	return {
		sent,
		repos: createApiRepos({
			baseUrl: 'https://api.example.org',
			fetch: backendFetch(backend, sent),
			clock
		})
	};
}

function repos(items: unknown[], total = items.length, activity = { users: 0, messages: 0 }) {
	return reposFor({ items, total, activity });
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
		const { sent, repos: r } = repos([
			narrative({}),
			narrative({
				id: 'n2',
				verdict: 'mixed',
				harm: 2,
				status: 'expert_verified',
				created_at: '2026-09-01T00:00:00Z'
			}),
			narrative({ id: 'n3', verdict: 'supported', harm: 1 })
		]);
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
		expect(s.activeNarratives).toEqual({ count: 3, new: 2 });
		expect(s.highHarm.awaitingReview).toBe(1);
		expect(s.creators).toEqual([]);
	});

	it('reads every page so all counts cover the same narratives', async () => {
		const items = Array.from({ length: 250 }, (_, i) =>
			narrative({
				id: `n${i}`,
				verdict: i % 2 === 0 ? 'contradicted' : 'supported',
				harm: 5,
				status: 'needs_check'
			})
		);
		const { sent, repos: r } = repos(items);
		const result = await r.overview.getSnapshot(
			{ accountId: 'a' },
			{ range: '24h', engagement: 'all' }
		);
		if (!result.ok) throw new Error('expected ok');
		const s = result.data;

		const pages = sent.filter((request) => request.includes('/narratives'));
		expect(pages).toHaveLength(3);
		expect(pages.some((request) => request.includes('page=3'))).toBe(true);
		expect(s.activeNarratives).toEqual({ count: 250, new: 250 });
		expect(s.rated).toEqual({ false: 125, misleading: 0, unverified: 0, accurate: 125 });
		expect(s.narratives).toHaveLength(125);
		expect(s.highHarm.awaitingReview).toBe(250);
	});

	it('fails when a later page cannot be read', async () => {
		const items = Array.from({ length: 150 }, (_, i) => narrative({ id: `n${i}` }));
		const { repos: r } = reposFor({
			items,
			respond: (url) => (url.searchParams.get('page') === '2' ? failed(500) : undefined)
		});
		expect(
			await r.overview.getSnapshot({ accountId: 'a' }, { range: '7d', engagement: 'all' })
		).toMatchObject({ ok: false, status: 500 });
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

	describe('when only one of the backend calls fails', () => {
		const ENDPOINT = {
			stats: '/conversations/stats',
			narratives: '/narratives',
			volume: '/conversations/volume',
			platforms: '/claims/platforms'
		} as const;

		const failingRepos = (failing: keyof typeof ENDPOINT, status: number) =>
			reposFor({
				respond: (url) => (url.pathname.endsWith(ENDPOINT[failing]) ? failed(status) : undefined)
			}).repos;

		it.each([
			['user activity', 'stats', 403],
			['narratives', 'narratives', 500],
			['message volume', 'volume', 502],
			['claims by platform', 'platforms', 500]
		] as const)('fails when the %s cannot be read', async (_name, failing, status) => {
			const r = failingRepos(failing, status);
			expect(
				await r.overview.getSnapshot({ accountId: 'a' }, { range: '7d', engagement: '7d' })
			).toMatchObject({ ok: false, status });
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

	describe('message volume', () => {
		const messagesOf = async (volume: unknown) => {
			const { repos: r } = reposFor({ volume });
			const result = await r.overview.getSnapshot(
				{ accountId: 'a' },
				{ range: '24h', engagement: 'all' }
			);
			if (!result.ok) throw new Error('expected ok');
			return result.data.messages;
		};

		it('maps the current window, the previous one and the trend', async () => {
			expect(await messagesOf({ current: 12, previous: 8, trend: [5, 7] })).toEqual({
				current: 12,
				previous: 8,
				trend: [5, 7]
			});
		});

		it('counts a field the backend left out as zero', async () => {
			expect(await messagesOf({})).toEqual({ current: 0, previous: 0, trend: [] });
		});

		it.each(['24h', '7d', '30d'] as const)('asks the backend for the %s window', async (range) => {
			const { sent, repos: r } = reposFor({});
			await r.overview.getSnapshot({ accountId: 'a' }, { range, engagement: 'all' });
			expect(sent).toContain(`/api/v1/conversations/volume?range=${range}`);
		});
	});

	describe('claims by platform', () => {
		const countsOf = async (platforms: unknown) => {
			const { repos: r } = reposFor({ platforms });
			const result = await r.overview.getSnapshot(
				{ accountId: 'a' },
				{ range: '24h', engagement: 'all' }
			);
			if (!result.ok) throw new Error('expected ok');
			return result.data.platformCounts;
		};

		it('maps the platforms the dashboard knows', async () => {
			expect(
				await countsOf([
					{ platform: 'whatsapp', claims: 10 },
					{ platform: 'tiktok', claims: 5 }
				])
			).toEqual([
				{ platform: 'whatsapp', count: 10 },
				{ platform: 'tiktok', count: 5 }
			]);
		});

		it('counts a platform the dashboard has no slice for as other, and sums them', async () => {
			expect(
				await countsOf([
					{ platform: 'instagram', claims: 3 },
					{ platform: 'myspace', claims: 2 }
				])
			).toEqual([{ platform: 'other', count: 5 }]);
		});

		it('sums repeated platforms into one slice', async () => {
			expect(
				await countsOf([
					{ platform: 'whatsapp', claims: 10 },
					{ platform: 'whatsapp', claims: 1 }
				])
			).toEqual([{ platform: 'whatsapp', count: 11 }]);
		});

		it('drops platforms without claims, also when the count is left out', async () => {
			expect(
				await countsOf([
					{ platform: 'x', claims: 0 },
					{ platform: 'facebook' },
					{ platform: 'tiktok', claims: 4 }
				])
			).toEqual([{ platform: 'tiktok', count: 4 }]);
		});

		it('orders the slices like PLATFORMS, not like the backend', async () => {
			const counts = await countsOf([
				{ platform: 'other', claims: 9 },
				{ platform: 'x', claims: 8 },
				{ platform: 'tiktok', claims: 7 },
				{ platform: 'facebook', claims: 6 },
				{ platform: 'whatsapp', claims: 5 }
			]);
			expect(counts.map((item) => item.platform)).toEqual([...PLATFORMS]);
		});

		it('is empty when no claims came in', async () => {
			expect(await countsOf([])).toEqual([]);
		});

		it.each(['24h', '7d', '30d'] as const)('asks the backend for the %s window', async (range) => {
			const { sent, repos: r } = reposFor({});
			await r.overview.getSnapshot({ accountId: 'a' }, { range, engagement: 'all' });
			expect(sent).toContain(`/api/v1/claims/platforms?range=${range}`);
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
