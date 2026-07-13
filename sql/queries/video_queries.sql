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