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
	CreateTimeISO time.Time `json:"createTimeISO" validate:"required"`

	// LocationCreated is the location where the video was created.
	LocationCreated string `json:"locationCreated"`

	// PlayCount is the number of times the video has been played.
	PlayCount int64 `json:"playCount" validate:"min=0"`

	// DiggCount is the number of likes the video has received.
	DiggCount int64 `json:"diggCount" validate:"min=0"`

	// SubmittedVideoURL is the URL of the submitted video.
	SubmittedVideoURL string `json:"submittedVideoUrl" validate:"required,url"`

	// MediaUrls is a list of media URLs associated with the video, which must contain at least one valid URL.
	MediaUrls []string `json:"mediaUrls" validate:"required,min=1,dive,url"`

	// AuthorMeta contains metadata about the author of the video.
	AuthorMeta AuthorMeta `json:"authorMeta" validate:"required"`

	// VideoMeta contains metadata about the video itself.
	VideoMeta VideoMeta `json:"videoMeta" validate:"required"`

	// Hashtags contains the list of hashtags extracted from the video.
	Hashtags []ApifyHashtag `json:"hashtags"`
}

// AuthorMeta represents metadata about the author of a TikTok video.
type AuthorMeta struct {
	// Name is the username of the content creator.
	Name string `json:"name" validate:"required"`

	// ProfileURL is the URL to the content creator's profile.
	ProfileURL string `json:"profileUrl" validate:"required,url"`
}

// VideoMeta represents metadata about a TikTok video.
type VideoMeta struct {
	// CoverUrl is the URL to the cover image of the video.
	CoverUrl string `json:"coverUrl" validate:"required,url"`
}
