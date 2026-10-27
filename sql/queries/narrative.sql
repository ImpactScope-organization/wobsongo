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
