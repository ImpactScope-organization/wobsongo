BEGIN;

DELETE FROM videos WHERE origin <> 'whatsapp';

DROP INDEX IF EXISTS idx_videos_status;
DROP INDEX IF EXISTS idx_videos_influencer_id;
DROP INDEX IF EXISTS uq_videos_type_external_id;

ALTER TABLE videos
    DROP COLUMN IF EXISTS analyzed_at,
    DROP COLUMN IF EXISTS discovered_via_keyword,
    DROP COLUMN IF EXISTS error,
    DROP COLUMN IF EXISTS failed_stage,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS origin,
    DROP COLUMN IF EXISTS media_url,
    DROP COLUMN IF EXISTS collect_count,
    DROP COLUMN IF EXISTS share_count,
    DROP COLUMN IF EXISTS comment_count,
    DROP COLUMN IF EXISTS external_id,
    DROP COLUMN IF EXISTS influencer_id;

COMMIT;