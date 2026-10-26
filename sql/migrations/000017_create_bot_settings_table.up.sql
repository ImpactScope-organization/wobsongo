BEGIN;

CREATE TABLE bot_settings (
    jid VARCHAR(64) PRIMARY KEY,
    tone VARCHAR(20) CHECK (tone IN ('formal', 'academic', 'friendly')),
    length VARCHAR(20) CHECK (length IN ('short', 'medium', 'long', 'expert')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;