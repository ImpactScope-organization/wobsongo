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