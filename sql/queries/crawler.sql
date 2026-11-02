-- name: GetCrawlerSetting :one
-- The table holds a single row. It is created on first read, so no seed
-- migration is needed.
INSERT INTO crawler_setting (id) VALUES (1)
ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
RETURNING
    discovery_interval_hours,
    max_results_keyword,
    max_results_influencer,
    max_cost_per_run::float8 AS max_cost_per_run,
    updated_by,
    updated_at;

-- name: GetLastCrawlRunStartedAt :one
SELECT started_at FROM crawl_runs
WHERE kind = $1
ORDER BY started_at DESC
LIMIT 1;

-- name: FailStaleCrawlRuns :exec
UPDATE crawl_runs
SET status = 'failed',
    finished_at = NOW(),
    error = 'interrupted: the server stopped during this run'
WHERE kind = sqlc.arg(kind)
  AND status = 'running'
  AND started_at < NOW() - (sqlc.arg(stale_seconds)::float8 * INTERVAL '1 second');

-- name: CreateCrawlRun :one
INSERT INTO crawl_runs (kind, trigger, triggered_by, params)
VALUES ($1, $2, $3, $4)
RETURNING id, started_at;

-- name: FinishCrawlRun :exec
UPDATE crawl_runs
SET status = $2,
    finished_at = NOW(),
    video_found = $3,
    video_new = $4,
    error = $5
WHERE id = $1;