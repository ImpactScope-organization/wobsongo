-- name: UpsertTopicByName :one
INSERT INTO topics (
    name, description, is_active
) VALUES (
    $1, $2, $3
)
ON CONFLICT (lower(name)) DO UPDATE SET
    description = EXCLUDED.description,
    is_active = EXCLUDED.is_active,
    updated_at = NOW()
RETURNING *;

-- name: ListActiveTopics :many
SELECT * FROM topics WHERE is_active = true ORDER BY name;


-- name: PaginateTopics :many
SELECT * FROM topics
WHERE (sqlc.narg(is_active)::boolean IS NULL OR is_active = sqlc.narg(is_active)::boolean)
  AND (sqlc.narg(search)::text IS NULL
       OR strpos(lower(name), lower(sqlc.narg(search)::text)) > 0)
ORDER BY name
LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;

-- name: CountTopics :one
SELECT count(*) FROM topics
WHERE (sqlc.narg(is_active)::boolean IS NULL OR is_active = sqlc.narg(is_active)::boolean)
  AND (sqlc.narg(search)::text IS NULL
       OR strpos(lower(name), lower(sqlc.narg(search)::text)) > 0);

-- name: CountActiveTopics :one
SELECT count(*) FROM topics WHERE is_active;

-- name: GetTopicByID :one
SELECT * FROM topics WHERE id = $1;

-- name: LockActiveTopicGuard :exec
SELECT pg_advisory_xact_lock(hashtext('topics_active_guard'));

-- name: CreateTopic :one
INSERT INTO topics (name, description, is_active)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateTopic :one
UPDATE topics
SET name = sqlc.arg(name),
    description = sqlc.arg(description),
    is_active = sqlc.arg(is_active)::boolean,
    updated_at = NOW()
WHERE topics.id = sqlc.arg(id)
  AND (sqlc.arg(is_active)::boolean
       OR NOT topics.is_active
       OR EXISTS (
           SELECT 1 FROM topics AS other
           WHERE other.is_active AND other.id <> sqlc.arg(id)))
RETURNING *;