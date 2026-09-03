-- name: UpsertInfluencer :one
INSERT INTO influencers (
    platform, username, profile_url, nickname, bio, avatar_url, verified,
    followers_count, following_count, hearts_count, reported_video_count,
    source, discovered_via_keyword, discovered_from_video_url
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
ON CONFLICT (platform, username) DO UPDATE SET
    profile_url = EXCLUDED.profile_url,
    nickname = EXCLUDED.nickname,
    bio = EXCLUDED.bio,
    avatar_url = EXCLUDED.avatar_url,
    verified = EXCLUDED.verified,
    followers_count = EXCLUDED.followers_count,
    following_count = EXCLUDED.following_count,
    hearts_count = EXCLUDED.hearts_count,
    reported_video_count = EXCLUDED.reported_video_count,
    updated_at = NOW()
RETURNING id, created_at, updated_at;

-- name: GetInfluencerByUsername :one
SELECT * FROM influencers WHERE platform = $1 AND username = $2 LIMIT 1;

-- name: ListInfluencers :many
SELECT * FROM influencers ORDER BY username;

-- name: UpdateInfluencerAggregates :exec
UPDATE influencers
SET
    first_video_at = $1,
    last_video_at = $2,
    avg_upload_interval_hours = $3,
    tracked_video_count = $4,
    last_checked_at = NOW(),
    updated_at = NOW()
WHERE id = $5;