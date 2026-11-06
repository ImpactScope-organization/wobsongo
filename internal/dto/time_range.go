package dto

import "time"

// TimeRange is a reporting window of the dashboard's Overview page.
type TimeRange string

const (
	TimeRange24h TimeRange = "24h"
	TimeRange7d  TimeRange = "7d"
	TimeRange30d TimeRange = "30d"
)

// TimeRangeDTO holds the query parameter shared by the Overview endpoints.
type TimeRangeDTO struct {
	// Range is the reporting window. Left out, it is 24h.
	Range TimeRange `query:"range" validate:"omitempty,oneof=24h 7d 30d" example:"24h" enums:"24h,7d,30d"`
}

// Window is the length of the window, 24h when unset.
func (r TimeRange) Window() time.Duration {
	switch r {
	case TimeRange7d:
		return 7 * 24 * time.Hour
	case TimeRange30d:
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// Bucket is the size of one trend bucket: hourly for 24h, daily otherwise.
func (r TimeRange) Bucket() time.Duration {
	switch r {
	case TimeRange7d, TimeRange30d:
		return 24 * time.Hour
	default:
		return time.Hour
	}
}
