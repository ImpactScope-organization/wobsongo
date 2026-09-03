package model

import (
	"time"

	"github.com/google/uuid"
)

// InfluencerVideo represents a video tracked from an influencer's TikTok profile.
type InfluencerVideo struct {
	// ID is the unique identifier of the tracked video record.
	ID uuid.UUID `json:"id"`

	// InfluencerID is the ID of the influencer who posted the video.
	InfluencerID uuid.UUID `json:"influencer_id"`

	// TikTokVideoID is TikTok's unique video ID, used to prevent duplicate records.
	TikTokVideoID string `json:"tiktok_video_id"`

	// VideoURL is the URL of the TikTok video.
	VideoURL string `json:"video_url"`

	// Caption is the video caption.
	Caption string `json:"caption"`

	// LikeCount is the number of likes on the video.
	LikeCount int64 `json:"like_count"`

	// CommentCount is the number of comments on the video.
	CommentCount int64 `json:"comment_count"`

	// ShareCount is the number of times the video was shared.
	ShareCount int64 `json:"share_count"`

	// PlayCount is the number of views on the video.
	PlayCount int64 `json:"play_count"`

	// CollectCount is the number of times the video was saved.
	CollectCount int64 `json:"collect_count"`

	// Hashtags contains the hashtags used in the video.
	Hashtags []string `json:"hashtags"`

	// VideoCreatedAt is the time when the video was posted on TikTok.
	VideoCreatedAt time.Time `json:"video_created_at"`

	// CreatedAt is the time when the record was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the time when the record was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}
