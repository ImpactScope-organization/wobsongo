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

-- name: ListUserMessagesWithoutClaims :many
SELECT m.id, m.content, m.created_at
FROM conversation_messages m
WHERE m.role = 'user'
  AND m.created_at >= sqlc.arg('since')::timestamptz
  AND m.content !~* 'https?://'
  AND NOT EXISTS (SELECT 1 FROM claims c WHERE c.message_id = m.id)
ORDER BY m.created_at
LIMIT sqlc.arg('max_rows');

-- name: ListVideoTranscriptsWithoutClaims :many
SELECT s.content, s.created_at, u.id AS message_id
FROM conversation_messages s
JOIN LATERAL (
    SELECT id FROM conversation_messages
    WHERE jid = s.jid AND role = 'user' AND created_at <= s.created_at
    ORDER BY created_at DESC
    LIMIT 1
) u ON true
WHERE s.role = 'system'
  AND s.content LIKE 'The video transcript is ready:%'
  AND s.created_at >= sqlc.arg('since')::timestamptz
  AND NOT EXISTS (SELECT 1 FROM claims c WHERE c.message_id = u.id)
ORDER BY s.created_at
LIMIT sqlc.arg('max_rows');