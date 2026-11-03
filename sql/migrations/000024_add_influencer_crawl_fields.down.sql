BEGIN;

DROP INDEX IF EXISTS idx_influencers_next_check_at;

ALTER TABLE influencers
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS next_check_at,
    DROP COLUMN IF EXISTS empty_check_count;

COMMIT;