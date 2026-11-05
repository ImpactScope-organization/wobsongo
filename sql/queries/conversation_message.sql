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
  AND (sqlc.narg('days')::integer IS NULL
       OR created_at >= NOW() - make_interval(days => sqlc.narg('days')::integer));