package queue

import (
	"slices"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func TestScreenJobs_InsertOpts(t *testing.T) {
	tests := []struct {
		name            string
		opts            river.InsertOpts
		wantQueue       string
		wantMaxAttempts int
	}{
		{"ScreenVideosJob", ScreenVideosJob{}.InsertOpts(), QueueScreenDispatch, 1},
		{"ScreenVideoJob", ScreenVideoJob{}.InsertOpts(), QueueScreening, ScreenMaxAttempts},
	}

	required := []rivertype.JobState{
		rivertype.JobStateAvailable,
		rivertype.JobStatePending,
		rivertype.JobStateRunning,
		rivertype.JobStateScheduled,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.Queue != tt.wantQueue {
				t.Errorf("queue = %q, want %q", tt.opts.Queue, tt.wantQueue)
			}
			if tt.opts.MaxAttempts != tt.wantMaxAttempts {
				t.Errorf("max attempts = %d, want %d", tt.opts.MaxAttempts, tt.wantMaxAttempts)
			}
			if !tt.opts.UniqueOpts.ByArgs {
				t.Error("jobs must be unique by args")
			}

			states := tt.opts.UniqueOpts.ByState
			for _, state := range required {
				if !slices.Contains(states, state) {
					t.Errorf("unique states %v must contain %q", states, state)
				}
			}
			if slices.Contains(states, rivertype.JobStateCompleted) {
				t.Error("a completed job must not block a new one")
			}
		})
	}
}

func TestScreenJobs_Kind(t *testing.T) {
	if got := (ScreenVideosJob{}).Kind(); got != string(JobTypeScreenVideos) {
		t.Errorf("ScreenVideosJob kind = %q, want %q", got, JobTypeScreenVideos)
	}
	if got := (ScreenVideoJob{}).Kind(); got != string(JobTypeScreenVideo) {
		t.Errorf("ScreenVideoJob kind = %q, want %q", got, JobTypeScreenVideo)
	}
}
