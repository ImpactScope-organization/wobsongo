BEGIN;

ALTER TABLE influencers
    ADD COLUMN next_check_at TIMESTAMPTZ,
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN empty_check_count INTEGER NOT NULL DEFAULT 0;

-- Existing influencers are checked on the next monitor run.
UPDATE influencers SET next_check_at = NOW();

CREATE INDEX idx_influencers_next_check_at
    ON influencers (next_check_at) WHERE is_active;

COMMIT;