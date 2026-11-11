import {
    HIGH_HARM_THRESHOLD,
    type HarmScore,
    type NarrativeSummary,
    type RatingOutcome,
    type ReviewStatus,
    type Verdict
} from '#lib/domain/narrative.js';
import type {
    ContentCreator,
    CreatorPlatform,
    EngagementWindow,
    OverviewSnapshot,
    UserQuestionStats
} from '#lib/domain/overview.js';
import type { TimeRange } from '#lib/domain/time-range.js';
import { ok, type Result } from '#lib/server/result.js';
import type { OverviewRepo } from '../../services/overview.js';
import { call, createApiClient, type Schemas } from './client.js';
import type { ApiOptions } from './index.js';
import { PLATFORMS, type Platform, type PlatformCount } from '#lib/domain/platform.js';

type ApiNarrative = Schemas['model.Narrative'];
type ApiUserActivity = Schemas['model.UserActivity'];
type ApiMessageVolume = Schemas['model.MessageVolume'];
type ApiPlatformCount = Schemas['model.ClaimPlatformCount'];
type ApiInfluencer = Schemas['model.Influencer'];

// The backend's page-size cap. Overview reads every page of active narratives,
// so all of its counts describe the same set.
const NARRATIVE_PAGE_SIZE = 100;

// How many content creators the Overview list shows.
const CREATOR_LIMIT = 10;

const CREATOR_PLATFORMS: readonly CreatorPlatform[] = ['tiktok', 'instagram', 'facebook', 'x'];

type NarrativePage = { items?: ApiNarrative[]; total_items: number };

const WINDOW_MS: Record<TimeRange, number> = {
    '24h': 24 * 60 * 60 * 1000,
    '7d': 7 * 24 * 60 * 60 * 1000,
    '30d': 30 * 24 * 60 * 60 * 1000
};

const STATUS: Record<NonNullable<ApiNarrative['status']>, ReviewStatus> = {
    needs_check: 'needs-check',
    auto_checked: 'auto-checked',
    expert_verified: 'expert-verified',
    not_in_kb: 'not-in-kb'
};

const OUTCOME: Record<string, RatingOutcome> = {
    contradicted: 'false',
    partially_supported: 'misleading',
    mixed: 'misleading',
    insufficient_evidence: 'unverified',
    supported: 'accurate'
};

const ENGAGEMENT_DAYS: Record<EngagementWindow, 7 | 15 | 30 | undefined> = {
    '7d': 7,
    '15d': 15,
    '30d': 30,
    all: undefined
};

function toUserQuestions(activity: ApiUserActivity): UserQuestionStats {
    return {
        users: activity.users ?? 0,
        questions: activity.messages ?? 0
    };
}

function toMessages(volume: ApiMessageVolume): OverviewSnapshot['messages'] {
    return {
        current: volume.current ?? 0,
        previous: volume.previous ?? 0,
        trend: volume.trend ?? []
    };
}

// Platforms the dashboard has no slice for are counted as "other".
function toPlatformCounts(items: ApiPlatformCount[]): PlatformCount[] {
    const totals = new Map<Platform, number>();
    for (const item of items) {
        const platform = PLATFORMS.find((value) => value === item.platform) ?? 'other';
        totals.set(platform, (totals.get(platform) ?? 0) + (item.claims ?? 0));
    }
    return PLATFORMS.filter((platform) => (totals.get(platform) ?? 0) > 0).map((platform) => ({
        platform,
        count: totals.get(platform) ?? 0
    }));
}

// Creators on a platform the dashboard has no icon for are left out. Verdict
// counts and risk level stay empty until post screening feeds them.
function toCreators(items: ApiInfluencer[]): ContentCreator[] {
    return items.flatMap((item) => {
        const platform = CREATOR_PLATFORMS.find((value) => value === item.platform);
        if (!platform) return [];

        const username = item.username ?? '';
        return [
            {
                id: item.id ?? '',
                name: item.nickname || username,
                handle: `@${username}`,
                platform,
                avatarUrl: item.avatar_url || undefined,
                profileUrl: item.profile_url || undefined,
                followers: item.followers_count ?? 0,
                postsCount: item.tracked_video_count ?? 0
            }
        ];
    });
}

function toOutcome(verdict: string | undefined): RatingOutcome {
    return (verdict && OUTCOME[verdict]) || 'unverified';
}

function toHarm(harm: number | undefined): HarmScore {
    const level = Math.min(5, Math.max(1, Math.round(harm ?? 1)));
    return level as HarmScore;
}

function toNarrativeSummary(narrative: ApiNarrative, verdict: Verdict): NarrativeSummary {
    return {
        id: narrative.id ?? '',
        title: narrative.statement ?? '',
        verdict,
        status: STATUS[narrative.status ?? 'needs_check'],
        domain: narrative.topic_name ?? '',
        harm: toHarm(narrative.harm),
        volume: 0,
        growthPct: 0,
        trend: []
    };
}

// Builds the Overview snapshot from the available API data.
function toSnapshot(
    page: NarrativePage,
    userQuestions: UserQuestionStats,
    messages: OverviewSnapshot['messages'],
    platformCounts: PlatformCount[],
    creators: ContentCreator[],
    range: TimeRange,
    engagement: EngagementWindow,
    now: Date
): OverviewSnapshot {
    const items = page.items ?? [];
    const since = now.getTime() - WINDOW_MS[range];

    const rated: Record<RatingOutcome, number> = {
        false: 0,
        misleading: 0,
        unverified: 0,
        accurate: 0
    };

    const narratives: NarrativeSummary[] = [];

    for (const item of items) {
        const outcome = toOutcome(item.verdict);
        rated[outcome] += 1;

        if (outcome !== 'accurate') {
            narratives.push(toNarrativeSummary(item, outcome));
        }
    }

    return {
        // TODO: The monitored region has no source yet.
        region: 'Burkina Faso',
        updatedAt: now,

        // Same count as the questions of the same window: user messages.
        messages,

        activeNarratives: {
            count: page.total_items,
            new: items.filter((item) => item.created_at && Date.parse(item.created_at) >= since).length
        },

        rated,
        medianDebunkMinutes: { current: 0, previous: 0 },
        narratives,
        creators,

        userQuestions,

        anomaly: null,
        platformCounts,

        highHarm: {
            // Assumption: high-harm narratives without expert verification
            // are considered to be awaiting review.
            awaitingReview: items.filter(
                (item) => toHarm(item.harm) >= HIGH_HARM_THRESHOLD && item.status !== 'expert_verified'
            ).length,
            fastTracked: 0
        }
    };
}

export function createApiOverviewRepo(options: ApiOptions): OverviewRepo {
    const api = createApiClient(options);

    const fetchNarrativePage = (page: number) =>
        call(
            api.GET('/api/v1/narratives', {
                params: {
                    query: {
                        is_active: true,
                        page,
                        per_page: NARRATIVE_PAGE_SIZE
                    }
                }
            }),
            (data): NarrativePage => data
        );

    // Reads every page of active narratives. The first page gives the total,
    // the remaining pages are then fetched in parallel.
    async function fetchActiveNarratives(): Promise<Result<NarrativePage>> {
        const first = await fetchNarrativePage(1);
        if (!first.ok) return first;

        const totalItems = first.data.total_items;
        const pageCount = Math.ceil(totalItems / NARRATIVE_PAGE_SIZE);
        const items = [...(first.data.items ?? [])];

        const rest = await Promise.all(
            Array.from({ length: Math.max(0, pageCount - 1) }, (_, i) => fetchNarrativePage(i + 2))
        );
        for (const page of rest) {
            if (!page.ok) return page;
            items.push(...(page.data.items ?? []));
        }

        // Derive every count from the list actually read, so they stay consistent
        // even if narratives changed between the page requests.
        return ok({ items, total_items: items.length });
    }

    return {
        // The backend has no time filter for active narratives.
        // The selected range only determines what counts as a new narrative.
        getSnapshot: async (_scope, { range, engagement }) => {
            const [narratives, activity, messages, platforms, creators] = await Promise.all([
                fetchActiveNarratives(),
                call(
                    api.GET('/api/v1/conversations/stats', {
                        params: { query: { days: ENGAGEMENT_DAYS[engagement] } }
                    }),
                    toUserQuestions
                ),
                call(api.GET('/api/v1/conversations/volume', { params: { query: { range } } }), toMessages),
                call(
                    api.GET('/api/v1/claims/platforms', { params: { query: { range } } }),
                    toPlatformCounts
                ),
                call(
                    api.GET('/api/v1/influencers', {
                        params: { query: { sort: 'posts', is_active: true, per_page: CREATOR_LIMIT } }
                    }),
                    (data) => toCreators(data.items ?? [])
                )
            ]);

            if (!narratives.ok) return narratives;
            if (!activity.ok) return activity;
            if (!messages.ok) return messages;
            if (!platforms.ok) return platforms;
            if (!creators.ok) return creators;

            return ok(
                toSnapshot(
                    narratives.data,
                    activity.data,
                    messages.data,
                    platforms.data,
                    creators.data,
                    range,
                    engagement,
                    options.clock.now()
                )
            );
        }
    };
}