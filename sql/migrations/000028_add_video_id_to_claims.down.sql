BEGIN;

DROP INDEX IF EXISTS idx_claims_video_id;
DROP INDEX IF EXISTS uq_claims_video_id_text;

ALTER TABLE claims DROP COLUMN IF EXISTS video_id;

COMMIT;