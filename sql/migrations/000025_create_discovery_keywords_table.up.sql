BEGIN;

CREATE TABLE discovery_keywords (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    keyword TEXT NOT NULL,
    language INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (keyword, language)
);

COMMIT;