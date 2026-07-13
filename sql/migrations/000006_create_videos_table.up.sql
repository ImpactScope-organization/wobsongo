BEGIN;

CREATE TABLE videos (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    video_url TEXT NOT NULL UNIQUE,
    author_username VARCHAR(100) NOT NULL,
    author_profile_url TEXT NOT NULL DEFAULT '',
    caption TEXT,
    play_count BIGINT DEFAULT 0,
    like_count BIGINT DEFAULT 0,
    thumbnail_url TEXT NOT NULL DEFAULT '',
    location_created VARCHAR(10),
    video_created_at TIMESTAMPTZ,
    transcription_text TEXT,
    hashtags TEXT[],
    video_type VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_videos_author_username ON videos(author_username);
CREATE INDEX idx_videos_video_type ON videos(video_type);

COMMIT;