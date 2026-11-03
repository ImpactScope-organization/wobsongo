BEGIN;

ALTER TABLE videos
    ADD COLUMN influencer_id UUID REFERENCES influencers(id) ON DELETE SET NULL,
    ADD COLUMN external_id VARCHAR(64),
    ADD COLUMN comment_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN share_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN collect_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN media_url TEXT,
    ADD COLUMN origin VARCHAR(20) NOT NULL DEFAULT 'whatsapp',
    ADD COLUMN status VARCHAR(20),
    ADD COLUMN failed_stage VARCHAR(30),
    ADD COLUMN error TEXT,
    ADD COLUMN discovered_via_keyword TEXT,
    ADD COLUMN analyzed_at TIMESTAMPTZ;

-- Backfill external_id for existing videos from the /video/<digits> URL
-- pattern. Short links (vm.tiktok.com/...) don't match, so their
-- external_id stays NULL.
UPDATE videos
SET external_id = substring(video_url FROM '/video/(\d+)')
WHERE video_type = 'tiktok';

CREATE UNIQUE INDEX uq_videos_type_external_id
    ON videos (video_type, external_id) WHERE external_id IS NOT NULL;
CREATE INDEX idx_videos_influencer_id ON videos (influencer_id);
CREATE INDEX idx_videos_status ON videos (status) WHERE status IS NOT NULL;

-- Videos already present in both tables: just link them to the influencer.
UPDATE videos v
SET influencer_id = iv.influencer_id,
    comment_count = iv.comment_count,
    share_count   = iv.share_count,
    collect_count = iv.collect_count
FROM influencer_videos iv
WHERE v.video_type = 'tiktok' AND v.external_id = iv.tiktok_video_id;

-- Copy the rest as new rows.
INSERT INTO videos (
    video_url, author_username, author_profile_url, caption,
    play_count, like_count, comment_count, share_count, collect_count,
    hashtags, video_created_at, video_type, influencer_id, external_id,
    origin, status, created_at, updated_at
)
SELECT
    iv.video_url, i.username, i.profile_url, iv.caption,
    iv.play_count, iv.like_count, iv.comment_count, iv.share_count, iv.collect_count,
    iv.hashtags, iv.video_created_at, i.platform, iv.influencer_id, iv.tiktok_video_id,
    'monitor', 'discovered', iv.created_at, iv.updated_at
FROM influencer_videos iv
JOIN influencers i ON i.id = iv.influencer_id
ON CONFLICT DO NOTHING;

COMMIT;