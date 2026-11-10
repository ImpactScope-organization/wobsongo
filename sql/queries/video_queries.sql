-- name: CreateVideos :one
INSERT INTO videos (
    video_url, 
    author_username, 
    author_profile_url, 
    caption,
    play_count, 
    like_count, 
    thumbnail_url, 
    location_created,
    video_created_at, 
    video_type,
    hashtags
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
ON CONFLICT (video_url) DO UPDATE SET
    play_count = EXCLUDED.play_count,
    like_count = EXCLUDED.like_count,
    updated_at = NOW()
RETURNING id, created_at, updated_at;

-- name: UpdateVideoTranscription :exec
UPDATE videos
SET 
    transcription_text = $1,
    updated_at = NOW()
WHERE id = $2;

-- name: GetVideoByURL :one
SELECT * FROM videos WHERE video_url = $1 LIMIT 1;


-- name: UpsertDiscoveredVideo :one
-- Videos from discovery/monitoring are identified by (video_type, external_id).
-- origin and status are only set on insert, so a video that came from
-- WhatsApp keeps its origin and stays out of the pipeline.
INSERT INTO videos (
    video_url, author_username, author_profile_url, caption,
    play_count, like_count, comment_count, share_count, collect_count,
    hashtags, video_created_at, video_type, influencer_id, external_id,
    origin, status, discovered_via_keyword
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
    'discovered', $16
)
ON CONFLICT (video_type, external_id) WHERE external_id IS NOT NULL DO UPDATE SET
    play_count = EXCLUDED.play_count,
    like_count = EXCLUDED.like_count,
    comment_count = EXCLUDED.comment_count,
    share_count = EXCLUDED.share_count,
    collect_count = EXCLUDED.collect_count,
    influencer_id = COALESCE(videos.influencer_id, EXCLUDED.influencer_id),
    updated_at = NOW()
RETURNING id, created_at, updated_at, (xmax = 0)::boolean AS is_new;

-- name: ListVideoCreatedAtByInfluencerID :many
SELECT video_created_at FROM videos
WHERE influencer_id = sqlc.arg(influencer_id)::uuid
  AND video_created_at IS NOT NULL
ORDER BY video_created_at ASC;

-- name: ListDiscoveredVideos :many
-- Returns the videos still waiting for screening, oldest first. 
SELECT id, caption, hashtags
FROM videos
WHERE status = 'discovered'
  AND id <> ALL(sqlc.arg(exclude_ids)::uuid[])
ORDER BY created_at ASC
LIMIT sqlc.arg(batch_size)::int;

-- name: SetVideoScreenStatus :execrows
UPDATE videos
SET status = sqlc.arg(status)::varchar, updated_at = NOW()
WHERE id = sqlc.arg(id)::uuid
  AND status = 'discovered';

-- name: GetVideoForScreening :one
SELECT id, caption, hashtags
FROM videos
WHERE id = sqlc.arg(id)::uuid
  AND status = 'discovered';