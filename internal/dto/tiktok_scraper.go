package dto

import "time"

// TikTokScraperItem represents a record returned by the Apify TikTok Scraper actor.
type TikTokScraperItem struct {
	// ID is the unique TikTok video ID.
	ID string `json:"id"`

	// Text is the video caption.
	Text string `json:"text"`

	// CreateTimeISO is the video's creation time.
	CreateTimeISO time.Time `json:"createTimeISO"`

	// WebVideoURL is the URL of the TikTok video.
	WebVideoURL string `json:"webVideoUrl"`

	// DiggCount is the number of likes.
	DiggCount int64 `json:"diggCount"`

	// ShareCount is the number of shares.
	ShareCount int64 `json:"shareCount"`

	// PlayCount is the number of views.
	PlayCount int64 `json:"playCount"`

	// CommentCount is the number of comments.
	CommentCount int64 `json:"commentCount"`

	// CollectCount is the number of saves.
	CollectCount int64 `json:"collectCount"`

	// Hashtags contains the hashtags used in the video.
	Hashtags []TikTokHashtag `json:"hashtags"`

	// AuthorMeta contains the video's author information.
	AuthorMeta TikTokAuthorMeta `json:"authorMeta"`
}

// TikTokHashtag represents a hashtag attached to a TikTok video.
type TikTokHashtag struct {
	// Name is the hashtag name.
	Name string `json:"name"`
}

// TikTokAuthorMeta contains metadata about a TikTok video author.
type TikTokAuthorMeta struct {
	// ID is the unique ID of the TikTok user.
	ID string `json:"id"`

	// Name is the author's TikTok username.
	Name string `json:"name"`

	// NickName is the author's display name.
	NickName string `json:"nickName"`

	// Verified indicates whether the author's account is verified.
	Verified bool `json:"verified"`

	// Signature is the author's profile bio.
	Signature string `json:"signature"`

	// Avatar is the URL of the author's profile picture.
	Avatar string `json:"avatar"`

	// Fans is the number of followers.
	Fans int64 `json:"fans"`

	// Following is the number of accounts the author follows.
	Following int64 `json:"following"`

	// Heart is the total number of likes received by the author's profile.
	Heart int64 `json:"heart"`

	// Video is the total number of videos posted by the author.
	Video int64 `json:"video"`
}
