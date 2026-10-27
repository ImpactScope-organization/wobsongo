BEGIN;

CREATE TABLE claims (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    message_id UUID REFERENCES conversation_messages(id) ON DELETE SET NULL,
    text TEXT NOT NULL,
    topic_id UUID NOT NULL REFERENCES topics(id),
    language INTEGER NOT NULL DEFAULT 0,
    platform VARCHAR(20) NOT NULL DEFAULT 'whatsapp',
    format VARCHAR(20) NOT NULL DEFAULT 'text',
    occurred_at TIMESTAMPTZ NOT NULL,
    verdict INTEGER NOT NULL,
    severity INTEGER NOT NULL,
    high_risk BOOLEAN NOT NULL DEFAULT false,
    high_risk_reason TEXT NOT NULL DEFAULT '',
    recommend_medical_consult BOOLEAN NOT NULL DEFAULT false,
    reasoning TEXT NOT NULL DEFAULT '',
    brief_reasoning TEXT NOT NULL DEFAULT '',
    citations JSONB NOT NULL DEFAULT '[]',
    judged_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (message_id, text)
);

CREATE INDEX idx_claims_occurred_at ON claims (occurred_at DESC);
CREATE INDEX idx_claims_topic_id_occurred_at ON claims (topic_id, occurred_at DESC);

COMMIT;