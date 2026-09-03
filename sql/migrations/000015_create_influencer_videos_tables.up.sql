BEGIN;

CREATE TABLE influencer_videos (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    influencer_id UUID NOT NULL REFERENCES influencers(id) ON DELETE CASCADE,
    tiktok_video_id VARCHAR(64) NOT NULL UNIQUE,
    video_url TEXT NOT NULL,
    caption TEXT NOT NULL DEFAULT '',
    like_count BIGINT NOT NULL DEFAULT 0,
    comment_count BIGINT NOT NULL DEFAULT 0,
    share_count BIGINT NOT NULL DEFAULT 0,
    play_count BIGINT NOT NULL DEFAULT 0,
    collect_count BIGINT NOT NULL DEFAULT 0,
    hashtags TEXT[],
    video_created_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_influencer_videos_influencer_id ON influencer_videos(influencer_id);

COMMIT;