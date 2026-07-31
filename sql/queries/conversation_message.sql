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
