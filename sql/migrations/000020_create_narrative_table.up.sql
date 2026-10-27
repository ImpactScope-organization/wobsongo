BEGIN;

CREATE TABLE narratives (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    topic_id UUID NOT NULL REFERENCES topics(id),
    statement TEXT NOT NULL,
    harm SMALLINT,
    harm_validated BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'needs_check',
    is_active BOOLEAN NOT NULL DEFAULT true,
    statement_edited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    statement_edited_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_narratives_topic_id ON narratives (topic_id);

COMMIT;