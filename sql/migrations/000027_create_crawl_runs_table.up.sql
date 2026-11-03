BEGIN;

CREATE TABLE crawl_runs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    kind VARCHAR(20) NOT NULL CHECK (kind IN ('discover', 'monitor')),
    trigger VARCHAR(20) NOT NULL CHECK (trigger IN ('scheduled', 'manual')),
    triggered_by UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'succeeded', 'failed')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    video_found INTEGER NOT NULL DEFAULT 0,
    video_new INTEGER NOT NULL DEFAULT 0,
    cost NUMERIC(10, 4),
    params JSONB NOT NULL DEFAULT '{}',
    error TEXT
);

CREATE INDEX idx_crawl_runs_kind_started_at ON crawl_runs (kind, started_at DESC);

-- Prevents two runs of the same kind from running at once. If the server
-- dies mid-run the row stays 'running' and blocks the next run, so stuck
-- runs will need a cleanup.
CREATE UNIQUE INDEX uq_crawl_runs_one_running_per_kind
    ON crawl_runs (kind) WHERE status = 'running';

COMMIT;