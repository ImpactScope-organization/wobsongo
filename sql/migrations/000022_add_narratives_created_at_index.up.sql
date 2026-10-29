BEGIN;

CREATE INDEX idx_narratives_created_at_id ON narratives (created_at DESC, id DESC);

COMMIT;