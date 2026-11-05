package dto

// UserActivityDTO holds the query parameters of the user activity endpoint.
type UserActivityDTO struct {
	// Days counts only the last Days days. Left out, all time is counted.
	Days *int32 `query:"days" validate:"omitempty,oneof=7 15 30" example:"7"`
}
