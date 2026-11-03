package model

import (
	"time"

	"github.com/google/uuid"
)

// VideoOrigin tells how a video entered the system.
type VideoOrigin string

const (
	// VideoOriginWhatsApp means a user submitted the video through the bot.
	VideoOriginWhatsApp VideoOrigin = "whatsapp"

	// VideoOriginDiscovery means the video was found by a keyword search.
	VideoOriginDiscovery VideoOrigin = "discovery"

	// VideoOriginMonitor means the video was found while monitoring a creator.
	VideoOriginMonitor VideoOrigin = "monitor"
)

// VideoStatus is how far a discovered video has progressed in the pipeline.
// It is nil for videos submitted through WhatsApp, which skip the pipeline.
type VideoStatus string

const (
	VideoStatusDiscovered  VideoStatus = "discovered"
	VideoStatusScreenedOut VideoStatus = "screened_out"
	VideoStatusScreenedIn  VideoStatus = "screened_in"
	VideoStatusTranscribed VideoStatus = "transcribed"
	VideoStatusOutOfScope  VideoStatus = "out_of_scope"
	VideoStatusVerified    VideoStatus = "verified"
	VideoStatusClustered   VideoStatus = "clustered"
	VideoStatusFailed      VideoStatus = "failed"
)

// Video represents the schema for the videos table in the database.
type Video struct {
	// ID is the unique identifier for the video record.
	ID uuid.UUID `json:"id" validate:"required" format:"uuid"`

	// VideoURL is the link to the video (e.g., TikTok or Instagram reels).
	VideoURL string `db:"video_url" json:"video_url"`

	// AuthorUsername is the username of the content creator.
	AuthorUsername string `db:"author_username" json:"author_username"`

	// AuthorProfileURL is the link to the content creator's profile.
	AuthorProfileURL string `db:"author_profile_url" json:"author_profile_url"`

	// Caption is the description of the video content.
	Caption string `db:"caption" json:"caption"`

	// PlayCount is the number of the video plays.
	PlayCount int64 `db:"play_count" json:"play_count"`

	// LikeCount is the number of likes the video has received.
	LikeCount int64 `db:"like_count" json:"like_count"`

	// ThumbnailURL is the link to the video's thumbnail image.
	ThumbnailURL string `db:"thumbnail_url" json:"thumbnail_url"`

	// LocationCreated is the location where the video was created.
	LocationCreated string `db:"location_created" json:"location_created"`

	// VideoCreatedAt is the timestamp when the video was created.
	VideoCreatedAt time.Time `db:"video_created_at" json:"video_created_at"`

	// TranscriptionText is the text extracted from the video.
	TranscriptionText *string `db:"transcription_text" json:"transcription_text"`

	// VideoType indicates the type of video (e.g., TikTok, Instagram).
	VideoType string `db:"video_type" json:"video_type"`

	// Hashtags contains the list of hashtags extracted from the video.
	Hashtags []string `json:"hashtags"`

	// InfluencerID is the creator who published the video. Nil for videos
	// whose creator is not tracked.
	InfluencerID *uuid.UUID `json:"influencer_id,omitempty" format:"uuid"`

	// ExternalID is the platform's own video ID, used to prevent duplicates.
	// Nil when it could not be read from the URL.
	ExternalID *string `json:"external_id,omitempty"`

	// MediaURL is the downloadable link from Apify, set right before transcription.
	MediaURL *string `json:"media_url,omitempty"`

	// CommentCount is the number of comments on the video.
	CommentCount int64 `json:"comment_count"`

	// ShareCount is the number of times the video was shared.
	ShareCount int64 `json:"share_count"`

	// CollectCount is the number of times the video was saved.
	CollectCount int64 `json:"collect_count"`

	// Origin tells how the video entered the system.
	Origin VideoOrigin `json:"origin"`

	// Status is the video's position in the pipeline. Nil for WhatsApp videos.
	Status *VideoStatus `json:"status,omitempty"`

	// FailedStage is the pipeline stage that failed. Set only when Status is failed.
	FailedStage *string `json:"failed_stage,omitempty"`

	// Error is the failure detail. Set only when Status is failed.
	Error *string `json:"error,omitempty"`

	// DiscoveredViaKeyword is the keyword that surfaced the video.
	DiscoveredViaKeyword *string `json:"discovered_via_keyword,omitempty"`

	// AnalyzedAt is when the video's claims were analyzed and verified.
	AnalyzedAt *time.Time `json:"analyzed_at,omitempty"`

	// CreatedAt is the timestamp when the record was created in the database.
	CreatedAt time.Time `db:"created_at" json:"created_at"`

	// UpdatedAt is the timestamp when the record was last updated in the database.
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
