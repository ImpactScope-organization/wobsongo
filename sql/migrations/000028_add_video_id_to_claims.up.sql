BEGIN;

ALTER TABLE claims
    ADD COLUMN video_id UUID REFERENCES videos(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX uq_claims_video_id_text
    ON claims (video_id, text) WHERE video_id IS NOT NULL;
CREATE INDEX idx_claims_video_id ON claims (video_id) WHERE video_id IS NOT NULL;

COMMIT;