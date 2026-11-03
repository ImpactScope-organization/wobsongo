-- name: ListActiveDiscoveryKeywords :many
SELECT * FROM discovery_keywords WHERE is_active ORDER BY keyword;

-- name: MarkDiscoveryKeywordRun :exec
UPDATE discovery_keywords
SET last_run_at = NOW(), updated_at = NOW()
WHERE id = $1;


-- name: PaginateDiscoveryKeywords :many
SELECT * FROM discovery_keywords
WHERE (sqlc.narg(is_active)::boolean IS NULL OR is_active = sqlc.narg(is_active)::boolean)
  AND (sqlc.narg(search)::text IS NULL
       OR strpos(lower(keyword), lower(sqlc.narg(search)::text)) > 0)
ORDER BY keyword, language
LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;

-- name: CountDiscoveryKeywords :one
SELECT count(*) FROM discovery_keywords
WHERE (sqlc.narg(is_active)::boolean IS NULL OR is_active = sqlc.narg(is_active)::boolean)
  AND (sqlc.narg(search)::text IS NULL
       OR strpos(lower(keyword), lower(sqlc.narg(search)::text)) > 0);

-- name: GetDiscoveryKeyword :one
SELECT * FROM discovery_keywords WHERE id = $1;

-- name: CreateDiscoveryKeyword :one
INSERT INTO discovery_keywords (keyword, language, is_active)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateDiscoveryKeyword :one
UPDATE discovery_keywords
SET keyword = $2, language = $3, is_active = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDiscoveryKeyword :execrows
DELETE FROM discovery_keywords WHERE id = $1;