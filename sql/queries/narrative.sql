-- name: NarrativeExistsForTopic :one
SELECT EXISTS (
    SELECT 1 FROM narratives WHERE topic_id = $1 AND statement = $2
);

-- name: InsertNarrative :one
INSERT INTO narratives (
    topic_id, statement, harm, harm_validated, status, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ListNarrativesByTopic :many
SELECT * FROM narratives WHERE topic_id = $1 ORDER BY created_at;

-- name: InsertNarrativeClaim :exec
INSERT INTO narrative_claims (claim_id, narrative_id)
VALUES ($1, $2)
ON CONFLICT (claim_id, narrative_id) DO NOTHING;

-- name: GetNewestClaimForNarrative :one
SELECT c.* FROM claims c
JOIN narrative_claims nc ON nc.claim_id = c.id
WHERE nc.narrative_id = $1
ORDER BY c.occurred_at DESC
LIMIT 1;

-- name: UpdateNarrativeFromNewestClaim :exec
UPDATE narratives SET
    harm = CASE WHEN harm_validated THEN harm ELSE sqlc.narg('harm')::smallint END,
    status = CASE WHEN status = 'expert_verified' THEN status ELSE sqlc.arg('status')::varchar END,
    is_active = true,
    updated_at = NOW()
WHERE id = sqlc.arg('id');

-- name: LockTopicForClustering :exec
SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0));

-- name: DeactivateStaleNarratives :execrows
UPDATE narratives n SET is_active = false, updated_at = NOW()
WHERE n.is_active
  AND NOT EXISTS (
      SELECT 1
      FROM narrative_claims nc
      JOIN claims c ON c.id = nc.claim_id
      WHERE nc.narrative_id = n.id AND c.occurred_at >= $1
  );

-- name: GetNarrative :one
SELECT
    sqlc.embed(n),
    t.name AS topic_name,
    COALESCE(latest.verdict, -1)::integer AS verdict
FROM narratives n
JOIN topics t ON t.id = n.topic_id
LEFT JOIN LATERAL (
    SELECT c.verdict
    FROM narrative_claims nc
    JOIN claims c ON c.id = nc.claim_id
    WHERE nc.narrative_id = n.id
    ORDER BY c.occurred_at DESC
    LIMIT 1
) latest ON true
WHERE n.id = $1;

-- name: PaginateNarratives :many
SELECT
    sqlc.embed(n),
    t.name AS topic_name,
    COALESCE(latest.verdict, -1)::integer AS verdict
FROM narratives n
JOIN topics t ON t.id = n.topic_id
LEFT JOIN LATERAL (
    SELECT c.verdict
    FROM narrative_claims nc
    JOIN claims c ON c.id = nc.claim_id
    WHERE nc.narrative_id = n.id
    ORDER BY c.occurred_at DESC
    LIMIT 1
) latest ON true
WHERE (sqlc.narg('is_active')::boolean IS NULL OR n.is_active = sqlc.narg('is_active')::boolean)
  AND (sqlc.narg('status')::varchar IS NULL OR n.status = sqlc.narg('status')::varchar)
ORDER BY n.created_at DESC, n.id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountNarratives :one
SELECT count(*) FROM narratives n
WHERE (sqlc.narg('is_active')::boolean IS NULL OR n.is_active = sqlc.narg('is_active')::boolean)
  AND (sqlc.narg('status')::varchar IS NULL OR n.status = sqlc.narg('status')::varchar);
