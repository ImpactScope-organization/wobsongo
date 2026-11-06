-- name: InsertClaim :one
INSERT INTO claims (
    message_id, text, topic_id, language, platform, format, occurred_at,
    verdict, severity, high_risk, high_risk_reason, recommend_medical_consult,
    reasoning, brief_reasoning, citations, judged_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
ON CONFLICT (message_id, text) DO NOTHING
RETURNING *;

-- name: GetClaim :one
SELECT * FROM claims WHERE id = $1;

-- name: CountClaimsByPlatform :many
SELECT
    c.platform,
    COUNT(DISTINCT COALESCE(m.jid, c.id::text) || '|' || c.text)::bigint AS claims
FROM claims c
LEFT JOIN conversation_messages m ON m.id = c.message_id
WHERE c.occurred_at >= sqlc.arg('since')::timestamptz
GROUP BY c.platform
ORDER BY claims DESC, c.platform;