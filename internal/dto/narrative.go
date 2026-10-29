package dto

// NarrativeListDTO holds the query parameters of the narrative list endpoint.
type NarrativeListDTO struct {
	PaginationDTO

	// IsActive keeps only active (true) or inactive (false) narratives.
	IsActive *bool `query:"is_active" example:"true"`

	// Status keeps only narratives with this status. Left out, every status
	// is returned.
	Status string `query:"status" validate:"omitempty,oneof=needs_check auto_checked not_in_kb expert_verified" example:"auto_checked"`
}
