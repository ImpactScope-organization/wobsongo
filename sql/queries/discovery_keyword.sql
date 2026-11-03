-- name: ListActiveDiscoveryKeywords :many
SELECT * FROM discovery_keywords WHERE is_active ORDER BY keyword;

-- name: MarkDiscoveryKeywordRun :exec
UPDATE discovery_keywords
SET last_run_at = NOW(), updated_at = NOW()
WHERE id = $1;