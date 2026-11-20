package service

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

var analyzeVideoCreatedAt = time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

func analyzeFixture(transcript string) *model.Video {
	return &model.Video{TranscriptionText: &transcript, VideoCreatedAt: analyzeVideoCreatedAt}
}

func TestAnalyzeVideo_ChecksTheTranscriptWithAVideoSource(t *testing.T) {
	video := analyzeFixture("lemon water cures infections")
	store := newPipelineStore(model.VideoStatusTranscribed, video)
	checker := &pipelineClaimChecker{result: &ClaimCheckResult{InScope: true}}

	if err := NewAnalyzeService(store, checker).AnalyzeVideo(t.Context(), video.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(checker.reqs) != 1 {
		t.Fatalf("claim checks = %d, want 1", len(checker.reqs))
	}
	req := checker.reqs[0]
	if req.Text != "lemon water cures infections" {
		t.Errorf("checked text = %q, want the transcript", req.Text)
	}
	src := req.Source
	if src == nil {
		t.Fatal("source must be set, otherwise the claims are not stored")
	}
	if src.VideoID == nil || *src.VideoID != video.ID {
		t.Errorf("source video id = %v, want %v", src.VideoID, video.ID)
	}
	if src.MessageID != nil {
		t.Errorf("source message id = %v, want nil for a video", src.MessageID)
	}
	if src.Platform != model.ClaimPlatformTikTok || src.Format != string(model.ClaimFormatVideo) {
		t.Errorf("source = %s/%s, want tiktok/video", src.Platform, src.Format)
	}
	if !src.OccurredAt.Equal(analyzeVideoCreatedAt) {
		t.Errorf("occurred at = %v, want the time the video was published", src.OccurredAt)
	}
	if got := store.status[video.ID]; got != model.VideoStatusVerified {
		t.Errorf("status = %q, want verified", got)
	}
}

func TestAnalyzeVideo_OutOfScopeTranscript(t *testing.T) {
	video := analyzeFixture("pasta recipe")
	store := newPipelineStore(model.VideoStatusTranscribed, video)
	checker := &pipelineClaimChecker{result: &ClaimCheckResult{InScope: false}}

	if err := NewAnalyzeService(store, checker).AnalyzeVideo(t.Context(), video.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := store.status[video.ID]; got != model.VideoStatusOutOfScope {
		t.Errorf("status = %q, want out_of_scope", got)
	}
}

func TestAnalyzeVideo_NothingToCheckIsOutOfScope(t *testing.T) {
	blank := "  \n "
	tests := []struct {
		name  string
		video *model.Video
	}{
		{"no transcript", &model.Video{VideoCreatedAt: analyzeVideoCreatedAt}},
		{"blank transcript", &model.Video{TranscriptionText: &blank}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newPipelineStore(model.VideoStatusTranscribed, tt.video)
			checker := &pipelineClaimChecker{result: &ClaimCheckResult{InScope: true}}

			err := NewAnalyzeService(store, checker).AnalyzeVideo(t.Context(), tt.video.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(checker.reqs) != 0 {
				t.Errorf("claim checks = %d, want 0: there is nothing to send to the LLM",
					len(checker.reqs))
			}
			if got := store.status[tt.video.ID]; got != model.VideoStatusOutOfScope {
				t.Errorf("status = %q, want out_of_scope", got)
			}
		})
	}
}

func TestAnalyzeVideo_Errors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s *pipelineStore, c *pipelineClaimChecker)
	}{
		{
			"get failure",
			func(s *pipelineStore, _ *pipelineClaimChecker) { s.getErr = errPipelineBoom },
		},
		{
			"claim check failure",
			func(_ *pipelineStore, c *pipelineClaimChecker) { c.err = errPipelineBoom },
		},
		{
			"storing the status fails",
			func(s *pipelineStore, _ *pipelineClaimChecker) { s.analyzedErr = errPipelineBoom },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			video := analyzeFixture("claim")
			store := newPipelineStore(model.VideoStatusTranscribed, video)
			checker := &pipelineClaimChecker{result: &ClaimCheckResult{InScope: true}}
			tt.setup(store, checker)

			err := NewAnalyzeService(store, checker).AnalyzeVideo(t.Context(), video.ID)

			if !errors.Is(err, errPipelineBoom) {
				t.Fatalf("error = %v, want it to wrap the failure", err)
			}
			if got := store.status[video.ID]; got != model.VideoStatusTranscribed {
				t.Errorf("status = %q, want it to stay transcribed so the retry can run", got)
			}
		})
	}
}

func TestAnalyzeVideo_NilResultIsAnError(t *testing.T) {
	video := analyzeFixture("claim")
	store := newPipelineStore(model.VideoStatusTranscribed, video)
	checker := &pipelineClaimChecker{} // answers nil, nil

	err := NewAnalyzeService(store, checker).AnalyzeVideo(t.Context(), video.ID)

	if !errors.Is(err, errNoClaimResult) {
		t.Fatalf("error = %v, want it to wrap errNoClaimResult", err)
	}
	if got := store.status[video.ID]; got != model.VideoStatusTranscribed {
		t.Errorf("status = %q, want it to stay transcribed so the retry can run", got)
	}
}

func TestAnalyzeVideo_SkipsVideosThatAreNotTranscribed(t *testing.T) {
	verified := analyzeFixture("claim")
	store := newPipelineStore(model.VideoStatusVerified, verified)

	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{"unknown video", uuid.New()},
		{"video already analyzed", verified.ID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &pipelineClaimChecker{result: &ClaimCheckResult{InScope: true}}

			if err := NewAnalyzeService(
				store,
				checker,
			).AnalyzeVideo(t.Context(), tt.id); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(checker.reqs) != 0 {
				t.Errorf("claim checks = %d, want 0", len(checker.reqs))
			}
		})
	}
}

func TestAnalyzeVideo_AnalyzedByAnotherRun(t *testing.T) {
	video := analyzeFixture("claim")
	store := newPipelineStore(model.VideoStatusTranscribed, video)
	checker := &pipelineClaimChecker{
		result: &ClaimCheckResult{InScope: true},
		onCheck: func() {
			// Another run finishes the same video while this one checks the claims.
			store.status[video.ID] = model.VideoStatusOutOfScope
		},
	}

	if err := NewAnalyzeService(store, checker).AnalyzeVideo(t.Context(), video.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := store.status[video.ID]; got != model.VideoStatusOutOfScope {
		t.Errorf("status = %q, want the other run's result to stay", got)
	}
}

func TestMarkAnalyzeFailed(t *testing.T) {
	t.Run("MarksTheVideoAsFailed", func(t *testing.T) {
		video := analyzeFixture("claim")
		store := newPipelineStore(model.VideoStatusTranscribed, video)

		err := NewAnalyzeService(store, nil).
			MarkAnalyzeFailed(t.Context(), video.ID, errors.New("llm down"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := pipelineFailure{stage: model.VideoFailedStageAnalyze, reason: "llm down"}
		if got := store.failures[video.ID]; got != want {
			t.Errorf("failure = %+v, want %+v", got, want)
		}
	})

	t.Run("IgnoresAVideoThatMovedOn", func(t *testing.T) {
		video := analyzeFixture("claim")
		store := newPipelineStore(model.VideoStatusVerified, video)

		err := NewAnalyzeService(store, nil).
			MarkAnalyzeFailed(t.Context(), video.ID, errPipelineBoom)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := store.status[video.ID]; got != model.VideoStatusVerified {
			t.Errorf("status = %q, want it untouched", got)
		}
	})

	t.Run("StoreErrorIsReturned", func(t *testing.T) {
		video := analyzeFixture("claim")
		store := newPipelineStore(model.VideoStatusTranscribed, video)
		store.failErr = errPipelineBoom

		err := NewAnalyzeService(store, nil).
			MarkAnalyzeFailed(t.Context(), video.ID, errors.New("llm down"))

		if !errors.Is(err, errPipelineBoom) {
			t.Fatalf("error = %v, want it to wrap the store error", err)
		}
	})
}

func TestEnqueueTranscribedVideos(t *testing.T) {
	a, b, other := analyzeFixture("claim a"), analyzeFixture("claim b"), analyzeFixture("claim c")
	store := newPipelineStore(model.VideoStatusTranscribed, a, b, other)
	store.status[other.ID] = model.VideoStatusVerified
	checker := &pipelineClaimChecker{}

	count, err := NewAnalyzeService(store, checker).EnqueueTranscribedVideos(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := []uuid.UUID{a.ID, b.ID}; count != 2 || !slices.Equal(store.analyzeEnqueued, want) {
		t.Errorf("count = %d, analysis queued for %v, want %v", count, store.analyzeEnqueued, want)
	}
	if len(store.transcribeEnqueued) != 0 {
		t.Errorf("transcription queued for %v, want none", store.transcribeEnqueued)
	}
	if len(checker.reqs) != 0 {
		t.Errorf("claim checks = %d, want 0 (analyzing happens in the jobs)", len(checker.reqs))
	}
	if got := store.status[a.ID]; got != model.VideoStatusTranscribed {
		t.Errorf("status = %q, the sweep must not change it", got)
	}
}

func TestEnqueueTranscribedVideos_EnqueueErrorIsReturned(t *testing.T) {
	store := newPipelineStore(model.VideoStatusTranscribed, analyzeFixture("claim"))
	store.enqueueErr = errPipelineBoom

	count, err := NewAnalyzeService(store, &pipelineClaimChecker{}).
		EnqueueTranscribedVideos(t.Context())
	if !errors.Is(err, errPipelineBoom) {
		t.Fatalf("error = %v, want it to wrap the enqueue error", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}
