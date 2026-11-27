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
			"TranscribeSingleDiscoveredVideoJob",
			TranscribeSingleDiscoveredVideoJob{}.InsertOpts(),
			QueueTranscription,
			TranscribeMaxAttempts,
		},
		{
			"AnalyzeSingleVideoJob",
			AnalyzeSingleVideoJob{}.InsertOpts(),
			QueueAnalysis,
			AnalyzeMaxAttempts,
		},
		{
			"TranscribeMultipleDiscoveredVideoJob",
			TranscribeMultipleDiscoveredVideoJob{}.InsertOpts(),
			QueueTranscribeDispatch,
			1,
		},
		{
			"AnalyzeMultipleVideoJob",
			AnalyzeMultipleVideoJob{}.InsertOpts(),
			QueueAnalyzeDispatch,
			1,
		},
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
	if got := (TranscribeSingleDiscoveredVideoJob{}).Kind(); got != string(
		JobTypeTranscribeSingleDiscoveredVideo,
	) {
		t.Errorf("TranscribeSingleDiscoveredVideoJob kind = %q, want %q",
			got, JobTypeTranscribeSingleDiscoveredVideo)
	}
	if got := (AnalyzeSingleVideoJob{}).Kind(); got != string(JobTypeAnalyzeSingleVideo) {
		t.Errorf("AnalyzeSingleVideoJob kind = %q, want %q", got, JobTypeAnalyzeSingleVideo)
	}
	if got := (TranscribeMultipleDiscoveredVideoJob{}).Kind(); got != string(
		JobTypeTranscribeMultipleDiscoveredVideo,
	) {
		t.Errorf("TranscribeMultipleDiscoveredVideoJob kind = %q, want %q",
			got, JobTypeTranscribeMultipleDiscoveredVideo)
	}
	if got := (AnalyzeMultipleVideoJob{}).Kind(); got != string(JobTypeAnalyzeMultipleVideo) {
		t.Errorf("AnalyzeMultipleVideoJob kind = %q, want %q", got, JobTypeAnalyzeMultipleVideo)
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
		{"ScreenVideosJob", ScreenMultipleVideoJob{}.Kind()},
		{"ScreenVideoJob", ScreenSingleVideoJob{}.Kind()},
		{"TranscribeSingleDiscoveredVideoJob", TranscribeSingleDiscoveredVideoJob{}.Kind()},
		{"AnalyzeSingleVideoJob", AnalyzeSingleVideoJob{}.Kind()},
		{"TranscribeMultipleDiscoveredVideoJob", TranscribeMultipleDiscoveredVideoJob{}.Kind()},
		{"AnalyzeMultipleVideoJob", AnalyzeMultipleVideoJob{}.Kind()},
	}

	seen := make(map[string]string, len(jobs))
	for _, j := range jobs {
		if other, ok := seen[j.kind]; ok {
			t.Errorf("%s and %s share the kind %q", other, j.name, j.kind)
		}
		seen[j.kind] = j.name
	}
}

func TestVideoPipelineJobs_DispatchHasItsOwnQueue(t *testing.T) {
	tests := []struct {
		name     string
		dispatch river.InsertOpts
		work     river.InsertOpts
	}{
		{
			"transcribe",
			TranscribeMultipleDiscoveredVideoJob{}.InsertOpts(),
			TranscribeSingleDiscoveredVideoJob{}.InsertOpts(),
		},
		{"analyze", AnalyzeMultipleVideoJob{}.InsertOpts(), AnalyzeSingleVideoJob{}.InsertOpts()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dispatch.Queue == tt.work.Queue {
				t.Errorf("dispatch and work share the queue %q", tt.dispatch.Queue)
			}
		})
	}
}
