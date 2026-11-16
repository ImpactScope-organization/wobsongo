package dto

// KeywordListDTO holds the query parameters of the keyword list endpoint.
type KeywordListDTO struct {
	PaginationDTO

	// IsActive keeps only active (true) or inactive (false) keywords.
	IsActive *bool `query:"is_active" example:"true"`

	// Search keeps keywords that contain this text, ignoring case.
	Search string `query:"search" validate:"omitempty,max=100" example:"vaccin"`
}

// CreateKeywordRequest is the body for adding a discovery keyword.
type CreateKeywordRequest struct {
	// Keyword is the search text used on the platform.
	Keyword string `json:"keyword" validate:"required,notblank,min=2,max=100" example:"vaccin"`

	// Language is the language of the keyword, "en" or "fr".
	Language string `json:"language" validate:"required,oneof=en fr" example:"fr"`

	// IsActive defaults to true when left out.
	IsActive *bool `json:"is_active" example:"true"`
}

// UpdateKeywordRequest replaces a discovery keyword. All fields are required.
type UpdateKeywordRequest struct {
	Keyword  string `json:"keyword"   validate:"required,notblank,min=2,max=100" example:"vaccin"`
	Language string `json:"language"  validate:"required,oneof=en fr"            example:"fr"`
	IsActive *bool  `json:"is_active" validate:"required"                        example:"true"`
}

// TopicListDTO holds the query parameters of the topic list endpoint.
type TopicListDTO struct {
	PaginationDTO

	// IsActive keeps only active (true) or inactive (false) topics.
	IsActive *bool `query:"is_active" example:"true"`

	// Search keeps topics whose name contains this text.
	Search string `query:"search" validate:"omitempty,max=100" example:"health"`
}

// CreateTopicRequest is the body for adding a topic.
type CreateTopicRequest struct {
	// Name is the unique display name of the topic.
	Name string `json:"name" validate:"required,notblank,min=2,max=100" example:"Health"`

	// Description explains what the topic covers. The claim analyzer reads it.
	Description string `json:"description" validate:"max=1000" example:"Claims about diseases and vaccines"`

	// IsActive defaults to true when left out.
	IsActive *bool `json:"is_active" example:"true"`
}

// UpdateTopicRequest replaces a topic. Every field is required except the
// description, which may be empty.
type UpdateTopicRequest struct {
	Name        string `json:"name"        validate:"required,notblank,min=2,max=100" example:"Health"`
	Description string `json:"description" validate:"max=1000"                        example:"Claims about diseases and vaccines"`
	IsActive    *bool  `json:"is_active"   validate:"required"                        example:"true"`
}

// InfluencerListDTO holds the query parameters of the influencer list endpoint.
type InfluencerListDTO struct {
	PaginationDTO

	// IsActive keeps only monitored (true) or not monitored (false) influencers.
	IsActive *bool `query:"is_active" example:"true"`

	// Source keeps only influencers added by an admin or seed list ("seed"), or
	// found automatically ("discovered").
	Source string `query:"source" validate:"omitempty,oneof=seed discovered" example:"seed"`

	// Search keeps influencers whose username or nickname contains this text.
	Search string `query:"search" validate:"omitempty,max=100" example:"jane doe"`

	// Sort orders the list: "newest" (default), or "posts" for the influencers
	// with the most tracked posts first, then the most followers.
	Sort string `query:"sort" validate:"omitempty,oneof=newest posts" example:"posts"`
}

// CreateInfluencerRequest is the body for adding a TikTok influencer to monitor.
type CreateInfluencerRequest struct {
	// Username is the TikTok username, with or without a leading @.
	Username string `json:"username" validate:"required,tiktokusername" example:"@someone"`
}

// SetInfluencerActiveRequest turns monitoring of an influencer on or off.
type SetInfluencerActiveRequest struct {
	IsActive *bool `json:"is_active" validate:"required" example:"false"`
}

// CreatorListDTO holds the query parameters of the dashboard's creator list.
type CreatorListDTO struct {
	PaginationDTO
	TimeRangeDTO

	// IsActive keeps only monitored (true) or not monitored (false) influencers.
	IsActive *bool `query:"is_active" example:"true"`

	// Search keeps influencers whose username or nickname contains this text.
	Search string `query:"search" validate:"omitempty,max=100" example:"jane doe"`
}
