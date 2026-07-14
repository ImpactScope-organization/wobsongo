package dto

import (
	"time"
)

type ApifyHashtag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

// ApifyTikTokItem represents the structure of a TikTok metadata item as returned by the Apify API.
type ApifyTikTokItem struct {
	// Text is the caption of the TikTok video.
	Text string `json:"text"`

	// CreateTimeISO is the timestamp when the video was created.
	CreateTimeISO time.Time `json:"create_time_iso" validate:"required"`

	// LocationCreated is the location where the video was created.
	LocationCreated string `json:"location_created"`

	// PlayCount is the number of times the video has been played.
	PlayCount int64 `json:"play_count" validate:"min=0"`

	// DiggCount is the number of likes the video has received.
	DiggCount int64 `json:"digg_count" validate:"min=0"`

	// SubmittedVideoURL is the URL of the submitted video.
	SubmittedVideoURL string `json:"submitted_video_url" validate:"required,url"`

	// MediaUrls is a list of media URLs associated with the video, which must contain at least one valid URL.
	MediaUrls []string `json:"media_urls" validate:"required,min=1,dive,url"`

	// AuthorMetadata contains metadata about the author of the video.
	AuthorMetadata AuthorMetadata `json:"author_metadata" validate:"required"`

	// VideoMetadata contains metadata about the video itself.
	VideoMetadata VideoMetadata `json:"video_metadata" validate:"required"`

	// Hashtags contains the list of hashtags extracted from the video.
	Hashtags []ApifyHashtag `json:"hashtags"`
}

// AuthorMetadata represents metadata about the author of a TikTok video.
type AuthorMetadata struct {
	// Name is the username of the content creator.
	Name string `json:"name" validate:"required"`

	// ProfileURL is the URL to the content creator's profile.
	ProfileURL string `json:"profile_url" validate:"required,url"`
}

// VideoMetadata represents metadata about a TikTok video.
type VideoMetadata struct {
	// CoverUrl is the URL to the cover image of the video.
	CoverUrl string `json:"cover_url" validate:"required,url"`
}
