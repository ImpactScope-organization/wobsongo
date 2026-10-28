BEGIN;
 
CREATE TABLE narrative_claims (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    claim_id UUID NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
    narrative_id UUID NOT NULL REFERENCES narratives(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (claim_id, narrative_id)
);
 
CREATE INDEX idx_narrative_claims_narrative_id ON narrative_claims (narrative_id);
 
COMMIT;
