BEGIN;

CREATE TABLE conversation_messages (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    jid VARCHAR(64) NOT NULL,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Powers "last N messages for this jid" lookups (agent context window) —
-- always filtered by jid and ordered by recency.
CREATE INDEX idx_conversation_messages_jid_created_at
    ON conversation_messages (jid, created_at DESC);

COMMIT;
