BEGIN;

-- A single-row table: id is always 1.
CREATE TABLE crawler_setting (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    discovery_interval_hours INTEGER NOT NULL DEFAULT 24,
    max_results_keyword INTEGER NOT NULL DEFAULT 20 CHECK (max_results_keyword > 0),
    max_results_influencer INTEGER NOT NULL DEFAULT 10 CHECK (max_results_influencer > 0),
    max_cost_per_run NUMERIC(10, 2) NOT NULL DEFAULT 5.00,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;