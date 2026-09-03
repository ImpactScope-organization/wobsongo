BEGIN;
CREATE TABLE influencers (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    platform VARCHAR(20) NOT NULL DEFAULT 'tiktok',
    username VARCHAR(100) NOT NULL,
    profile_url TEXT NOT NULL DEFAULT '',
    nickname TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    verified BOOLEAN NOT NULL DEFAULT false,
    followers_count BIGINT NOT NULL DEFAULT 0,
    following_count BIGINT NOT NULL DEFAULT 0,
    hearts_count BIGINT NOT NULL DEFAULT 0,
    reported_video_count BIGINT NOT NULL DEFAULT 0,
    tracked_video_count BIGINT NOT NULL DEFAULT 0,
    source VARCHAR(20) NOT NULL DEFAULT 'seed',
    discovered_via_keyword TEXT,
    discovered_from_video_url TEXT,
    first_video_at TIMESTAMPTZ,
    last_video_at TIMESTAMPTZ,
    avg_upload_interval_hours DOUBLE PRECISION,
    last_checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (platform, username)
);

CREATE INDEX idx_influencers_source ON influencers(source);

COMMIT;