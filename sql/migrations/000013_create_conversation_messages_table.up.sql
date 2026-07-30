BEGIN;

CREATE TABLE conversation_messages (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    jid VARCHAR(64) NOT NULL,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversation_messages_jid_created_at
    ON conversation_messages (jid, created_at DESC);

COMMIT;
