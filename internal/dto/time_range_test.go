package dto

import (
	"testing"
	"time"
)

func TestTimeRange_WindowAndBucket(t *testing.T) {
	const day = 24 * time.Hour

	tests := []struct {
		name   string
		r      TimeRange
		window time.Duration
		bucket time.Duration
	}{
		{"24h", TimeRange24h, day, time.Hour},
		{"7d", TimeRange7d, 7 * day, day},
		{"30d", TimeRange30d, 30 * day, day},
		{"UnsetDefaultsTo24h", "", day, time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Window(); got != tt.window {
				t.Errorf("expected window %s, got %s", tt.window, got)
			}
			if got := tt.r.Bucket(); got != tt.bucket {
				t.Errorf("expected bucket %s, got %s", tt.bucket, got)
			}
			// The repo rejects a window that is not a whole number of buckets.
			if tt.r.Window()%tt.r.Bucket() != 0 {
				t.Errorf("window %s is not a multiple of bucket %s", tt.r.Window(), tt.r.Bucket())
			}
		})
	}
}
