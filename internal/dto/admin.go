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
