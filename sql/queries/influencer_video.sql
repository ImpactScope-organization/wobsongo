-- name: UpsertInfluencerVideo :one
INSERT INTO influencer_videos (
    influencer_id, tiktok_video_id, video_url, caption,
    like_count, comment_count, share_count, play_count, collect_count,
    hashtags, video_created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
ON CONFLICT (tiktok_video_id) DO UPDATE SET
    like_count = EXCLUDED.like_count,
    comment_count = EXCLUDED.comment_count,
    share_count = EXCLUDED.share_count,
    play_count = EXCLUDED.play_count,
    collect_count = EXCLUDED.collect_count,
    updated_at = NOW()
RETURNING id, created_at, updated_at, (xmax = 0)::boolean AS is_new;

-- name: ListInfluencerVideosByInfluencerID :many
SELECT * FROM influencer_videos WHERE influencer_id = $1 ORDER BY video_created_at ASC;