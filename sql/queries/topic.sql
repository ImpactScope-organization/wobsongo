-- name: UpsertTopicByName :one
INSERT INTO topics (
    name, description, is_active
) VALUES (
    $1, $2, $3
)
ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    is_active = EXCLUDED.is_active,
    updated_at = NOW()
RETURNING *;

-- name: ListActiveTopics :many
SELECT * FROM topics WHERE is_active = true ORDER BY name;