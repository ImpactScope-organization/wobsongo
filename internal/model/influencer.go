package model

import (
	"time"

	"github.com/google/uuid"
)

// InfluencerSource indicates how an influencer entered the system.
type InfluencerSource string

const (
	// InfluencerSourceSeed means the influencer was part of the initial seed list.
	InfluencerSourceSeed InfluencerSource = "seed"

	// InfluencerSourceDiscovered means the influencer was found automatically.
	InfluencerSourceDiscovered InfluencerSource = "discovered"
)

// Influencer represents a tracked TikTok influencer.
type Influencer struct {
	// ID is the unique identifier of the influencer.
	ID uuid.UUID `json:"id"`

	// Platform is the social media platform of the influencer.
	Platform string `json:"platform"`

	// Username is the influencer's username.
	Username string `json:"username"`

	// ProfileURL is the URL of the influencer's profile.
	ProfileURL string `json:"profile_url"`

	// Nickname is the influencer's display name.
	Nickname string `json:"nickname"`

	// Bio is the influencer's profile bio.
	Bio string `json:"bio"`

	// AvatarURL is the URL of the influencer's profile picture.
	AvatarURL string `json:"avatar_url"`

	// Verified indicates whether the influencer's account is verified.
	Verified bool `json:"verified"`

	// FollowersCount is the number of followers.
	FollowersCount int64 `json:"followers_count"`

	// FollowingCount is the number of accounts the influencer follows.
	FollowingCount int64 `json:"following_count"`

	// HeartsCount is the total number of likes received by the influencer.
	HeartsCount int64 `json:"hearts_count"`

	// ReportedVideoCount is the total number of videos reported by TikTok/Apify.
	ReportedVideoCount int64 `json:"reported_video_count"`

	// TrackedVideoCount is the number of videos captured by the monitoring system.
	TrackedVideoCount int64 `json:"tracked_video_count"`

	// Source indicates how the influencer was added to the system.
	Source InfluencerSource `json:"source"`

	// DiscoveredViaKeyword is the keyword that led to the influencer's discovery.
	DiscoveredViaKeyword string `json:"discovered_via_keyword,omitempty"`

	// DiscoveredFromVideoURL is the video URL where the influencer was discovered.
	DiscoveredFromVideoURL string `json:"discovered_from_video_url,omitempty"`

	// FirstVideoAt is the earliest tracked video time.
	FirstVideoAt *time.Time `json:"first_video_at,omitempty"`

	// LastVideoAt is the latest tracked video time.
	LastVideoAt *time.Time `json:"last_video_at,omitempty"`

	// AvgUploadIntervalHours is the average time between tracked uploads.
	AvgUploadIntervalHours *float64 `json:"avg_upload_interval_hours,omitempty"`

	// LastCheckedAt is the last time the influencer was checked for new videos.
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`

	// CreatedAt is the time when the record was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the time when the record was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}
