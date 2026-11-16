package queue

import (
	"slices"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func TestVideoPipelineJobs_InsertOpts(t *testing.T) {
	tests := []struct {
		name            string
		opts            river.InsertOpts
		wantQueue       string
		wantMaxAttempts int
	}{
		{
			"TranscribeDiscoveredVideoJob",
			TranscribeDiscoveredVideoJob{}.InsertOpts(),
			QueueTranscription,
			TranscribeMaxAttempts,
		},
		{"AnalyzeVideoJob", AnalyzeVideoJob{}.InsertOpts(), QueueAnalysis, AnalyzeMaxAttempts},
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

func TestVideoPipelineJobs_Kind(t *testing.T) {
	if got := (TranscribeDiscoveredVideoJob{}).Kind(); got != string(
		JobTypeTranscribeDiscoveredVideo,
	) {
		t.Errorf("TranscribeDiscoveredVideoJob kind = %q, want %q",
			got, JobTypeTranscribeDiscoveredVideo)
	}
	if got := (AnalyzeVideoJob{}).Kind(); got != string(JobTypeAnalyzeVideo) {
		t.Errorf("AnalyzeVideoJob kind = %q, want %q", got, JobTypeAnalyzeVideo)
	}
}


func TestJobKinds_AreUnique(t *testing.T) {
	jobs := []struct {
		name string
		kind string
	}{
		{"ExtractMediaDTO", ExtractMediaDTO{}.Kind()},
		{"TranscriptionJob", TranscriptionJob{}.Kind()},
		{"ClaimCheckJob", ClaimCheckJob{}.Kind()},
		{"ClusterClaimJob", ClusterClaimJob{}.Kind()},
		{"DeactivateNarrativesJob", DeactivateNarrativesJob{}.Kind()},
		{"DiscoverVideosJob", DiscoverVideosJob{}.Kind()},
		{"MonitorCreatorJob", MonitorCreatorJob{}.Kind()},
		{"ScreenVideosJob", ScreenVideosJob{}.Kind()},
		{"ScreenVideoJob", ScreenVideoJob{}.Kind()},
		{"TranscribeDiscoveredVideoJob", TranscribeDiscoveredVideoJob{}.Kind()},
		{"AnalyzeVideoJob", AnalyzeVideoJob{}.Kind()},
	}

	seen := make(map[string]string, len(jobs))
	for _, j := range jobs {
		if other, ok := seen[j.kind]; ok {
			t.Errorf("%s and %s share the kind %q", other, j.name, j.kind)
		}
		seen[j.kind] = j.name
	}
}
