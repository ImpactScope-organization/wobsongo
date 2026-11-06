-- name: InsertConversationMessage :one
INSERT INTO conversation_messages (
    jid, role, content
) VALUES (
    $1, $2, $3
)
RETURNING id, created_at;

-- name: GetRecentConversationMessages :many
SELECT * FROM conversation_messages
WHERE jid = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: CountUserActivity :one
SELECT
    COUNT(DISTINCT jid)::bigint AS users,
    COUNT(*)::bigint AS messages
FROM conversation_messages
WHERE role = 'user'
  AND created_at >= COALESCE(
        NOW() - make_interval(days => sqlc.narg('days')::integer),
        '-infinity'::timestamptz
      );


-- name: CountUserMessagesByBucket :many
WITH bounds AS (
    SELECT
        sqlc.arg('bucket_seconds')::float8 AS bucket_seconds,
        NOW() - make_interval(
            secs => sqlc.arg('bucket_seconds')::float8 * sqlc.arg('bucket_count')::integer * 2
        ) AS from_time
)
SELECT
    FLOOR(EXTRACT(EPOCH FROM (m.created_at - b.from_time))::float8 / b.bucket_seconds)::integer AS bucket,
    COUNT(*)::bigint AS messages
FROM conversation_messages m
CROSS JOIN bounds b
WHERE m.role = 'user'
  AND m.created_at >= b.from_time
  AND m.created_at < NOW()
GROUP BY 1
ORDER BY 1;