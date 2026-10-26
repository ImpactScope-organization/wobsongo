-- name: GetBotSettings :one
SELECT * FROM bot_settings WHERE jid = $1;

-- name: UpsertBotSettings :one
INSERT INTO bot_settings (
    jid, tone, length
) VALUES (
    $1, sqlc.narg('tone'), sqlc.narg('length')
)
ON CONFLICT (jid) DO UPDATE SET
    tone = COALESCE(EXCLUDED.tone, bot_settings.tone),
    length = COALESCE(EXCLUDED.length, bot_settings.length),
    updated_at = NOW()
RETURNING *;