package service

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

const testMediaURL = "https://api.apify.com/v2/key-value-stores/store/records/video.mp4"

func transcribeFixture() *model.Video {
	return &model.Video{VideoURL: "https://www.tiktok.com/@creator/video/123"}
}

func TestTranscribeVideo_ResolvesTranscribesAndQueuesAnalysis(t *testing.T) {
	video := transcribeFixture()
	store := newPipelineStore(model.VideoStatusScreenedIn, video)
	resolver := &pipelineMediaResolver{url: testMediaURL}
	transcriber := &pipelineTranscriber{transcript: "lemon water cures infections"}

	err := NewTranscribeService(store, resolver, transcriber).TranscribeVideo(t.Context(), video.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !slices.Equal(resolver.calls, []string{video.VideoURL}) {
		t.Errorf("resolver called with %v, want the video page url", resolver.calls)
	}
	if video.MediaURL == nil || *video.MediaURL != testMediaURL {
		t.Errorf("media url = %v, want it stored as %q", video.MediaURL, testMediaURL)
	}
	if !slices.Equal(transcriber.gotURLs, []string{testMediaURL}) {
		t.Errorf("asr got %v, want the resolved media url", transcriber.gotURLs)
	}
	if !slices.Equal(transcriber.gotFormats, []string{transcribeAudioFormat}) {
		t.Errorf("asr format = %v, want %q", transcriber.gotFormats, transcribeAudioFormat)
	}
	if got := store.status[video.ID]; got != model.VideoStatusTranscribed {
		t.Errorf("status = %q, want transcribed", got)
	}
	if got := store.transcripts[video.ID]; got != "lemon water cures infections" {
		t.Errorf("transcript = %q", got)
	}
	if !slices.Equal(store.analyzeEnqueued, []uuid.UUID{video.ID}) {
		t.Errorf("analysis queued for %v, want %v", store.analyzeEnqueued, video.ID)
	}
	if store.txCalls != 1 {
		t.Errorf("transactions = %d, want the status and the job in one", store.txCalls)
	}
}

func TestTranscribeVideo_UsesStoredMediaURL(t *testing.T) {
	video := transcribeFixture()
	stored := testMediaURL
	video.MediaURL = &stored
	store := newPipelineStore(model.VideoStatusScreenedIn, video)
	resolver := &pipelineMediaResolver{err: errPipelineBoom}
	transcriber := &pipelineTranscriber{transcript: "text"}

	err := NewTranscribeService(store, resolver, transcriber).TranscribeVideo(t.Context(), video.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolver.calls) != 0 {
		t.Errorf("resolver calls = %d, want 0: a stored url must not cost another Apify run",
			len(resolver.calls))
	}
	if !slices.Equal(transcriber.gotURLs, []string{testMediaURL}) {
		t.Errorf("asr got %v, want the stored media url", transcriber.gotURLs)
	}
}

func TestTranscribeVideo_EmptyTranscriptStillMovesOn(t *testing.T) {
	video := transcribeFixture()
	store := newPipelineStore(model.VideoStatusScreenedIn, video)
	resolver := &pipelineMediaResolver{url: testMediaURL}
	transcriber := &pipelineTranscriber{transcript: ""}

	err := NewTranscribeService(store, resolver, transcriber).TranscribeVideo(t.Context(), video.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := store.status[video.ID]; got != model.VideoStatusTranscribed {
		t.Errorf("status = %q, want transcribed (a video without speech is not an error)", got)
	}
	if len(store.analyzeEnqueued) != 1 {
		t.Errorf("analysis jobs = %d, want 1", len(store.analyzeEnqueued))
	}
}

func TestTranscribeVideo_Errors(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(s *pipelineStore, r *pipelineMediaResolver, tr *pipelineTranscriber)
		wantErr         error
		wantResolves    int
		wantTranscribes int
		wantMediaStored bool
	}{
		{
			name:    "get failure",
			setup:   func(s *pipelineStore, _ *pipelineMediaResolver, _ *pipelineTranscriber) { s.getErr = errPipelineBoom },
			wantErr: errPipelineBoom,
		},
		{
			name:         "resolve failure",
			setup:        func(_ *pipelineStore, r *pipelineMediaResolver, _ *pipelineTranscriber) { r.err = errPipelineBoom },
			wantErr:      errPipelineBoom,
			wantResolves: 1,
		},
		{
			name: "media unavailable",
			setup: func(_ *pipelineStore, r *pipelineMediaResolver, _ *pipelineTranscriber) {
				r.err = data.ErrMediaUnavailable
			},
			wantErr:      data.ErrMediaUnavailable,
			wantResolves: 1,
		},
		{
			name:         "storing the media url fails",
			setup:        func(s *pipelineStore, _ *pipelineMediaResolver, _ *pipelineTranscriber) { s.mediaErr = errPipelineBoom },
			wantErr:      errPipelineBoom,
			wantResolves: 1,
		},
		{
			// The media url is kept, so the retry does not pay for Apify again.
			name:            "asr failure keeps the media url",
			setup:           func(_ *pipelineStore, _ *pipelineMediaResolver, tr *pipelineTranscriber) { tr.err = errPipelineBoom },
			wantErr:         errPipelineBoom,
			wantResolves:    1,
			wantTranscribes: 1,
			wantMediaStored: true,
		},
		{
			name: "storing the transcript fails",
			setup: func(s *pipelineStore, _ *pipelineMediaResolver, _ *pipelineTranscriber) {
				s.transcribedErr = errPipelineBoom
			},
			wantErr:         errPipelineBoom,
			wantResolves:    1,
			wantTranscribes: 1,
			wantMediaStored: true,
		},
		{
			name: "queuing the analysis fails",
			setup: func(s *pipelineStore, _ *pipelineMediaResolver, _ *pipelineTranscriber) {
				s.enqueueErr = errPipelineBoom
			},
			wantErr:         errPipelineBoom,
			wantResolves:    1,
			wantTranscribes: 1,
			wantMediaStored: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			video := transcribeFixture()
			store := newPipelineStore(model.VideoStatusScreenedIn, video)
			resolver := &pipelineMediaResolver{url: testMediaURL}
			transcriber := &pipelineTranscriber{transcript: "text"}
			tt.setup(store, resolver, transcriber)

			err := NewTranscribeService(store, resolver, transcriber).
				TranscribeVideo(t.Context(), video.ID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want it to wrap %v", err, tt.wantErr)
			}
			if len(resolver.calls) != tt.wantResolves {
				t.Errorf("resolver calls = %d, want %d", len(resolver.calls), tt.wantResolves)
			}
			if len(transcriber.gotURLs) != tt.wantTranscribes {
				t.Errorf("asr calls = %d, want %d", len(transcriber.gotURLs), tt.wantTranscribes)
			}
			if stored := video.MediaURL != nil; stored != tt.wantMediaStored {
				t.Errorf("media url stored = %t, want %t", stored, tt.wantMediaStored)
			}
			if got := store.status[video.ID]; got != model.VideoStatusScreenedIn {
				t.Errorf("status = %q, want it to stay screened_in", got)
			}
			if _, ok := store.transcripts[video.ID]; ok {
				t.Error("a transcript must not be stored when the step fails")
			}
			if len(store.analyzeEnqueued) != 0 {
				t.Errorf("analysis jobs = %d, want 0", len(store.analyzeEnqueued))
			}
		})
	}
}

func TestTranscribeVideo_SkipsVideosThatAreNotScreenedIn(t *testing.T) {
	transcribed := transcribeFixture()
	store := newPipelineStore(model.VideoStatusTranscribed, transcribed)

	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{"unknown video", uuid.New()},
		{"video that already moved on", transcribed.ID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &pipelineMediaResolver{url: testMediaURL}
			transcriber := &pipelineTranscriber{}

			err := NewTranscribeService(store, resolver, transcriber).
				TranscribeVideo(t.Context(), tt.id)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(resolver.calls) != 0 || len(transcriber.gotURLs) != 0 {
				t.Error("a video that is not screened in must not cost an Apify run or an ASR call")
			}
		})
	}
}

func TestTranscribeVideo_AlreadyTranscribedByAnotherRun(t *testing.T) {
	video := transcribeFixture()
	store := newPipelineStore(model.VideoStatusScreenedIn, video)
	transcriber := &pipelineTranscriber{
		transcript: "text",
		onCall: func() {
			// Another run transcribes the same video while this one waits for the ASR.
			store.status[video.ID] = model.VideoStatusTranscribed
		},
	}

	err := NewTranscribeService(store, &pipelineMediaResolver{url: testMediaURL}, transcriber).
		TranscribeVideo(t.Context(), video.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.analyzeEnqueued) != 0 {
		t.Errorf("analysis jobs = %d, want 0: the other run already queued it",
			len(store.analyzeEnqueued))
	}
}

func TestMarkTranscribeFailed(t *testing.T) {
	t.Run("MarksTheVideoAsFailed", func(t *testing.T) {
		video := transcribeFixture()
		store := newPipelineStore(model.VideoStatusScreenedIn, video)

		err := NewTranscribeService(store, nil, nil).
			MarkTranscribeFailed(t.Context(), video.ID, errors.New("asr down"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := pipelineFailure{stage: model.VideoFailedStageTranscribe, reason: "asr down"}
		if got := store.failures[video.ID]; got != want {
			t.Errorf("failure = %+v, want %+v", got, want)
		}
		if got := store.status[video.ID]; got != model.VideoStatusFailed {
			t.Errorf("status = %q, want failed", got)
		}
	})

	t.Run("CutsALongReason", func(t *testing.T) {
		video := transcribeFixture()
		store := newPipelineStore(model.VideoStatusScreenedIn, video)
		cause := errors.New(strings.Repeat("x", maxFailureReasonLen+50))

		err := NewTranscribeService(store, nil, nil).
			MarkTranscribeFailed(t.Context(), video.ID, cause)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := len([]rune(store.failures[video.ID].reason)); got != maxFailureReasonLen {
			t.Errorf("reason length = %d, want %d", got, maxFailureReasonLen)
		}
	})

	t.Run("IgnoresAVideoThatMovedOn", func(t *testing.T) {
		video := transcribeFixture()
		store := newPipelineStore(model.VideoStatusTranscribed, video)

		err := NewTranscribeService(store, nil, nil).
			MarkTranscribeFailed(t.Context(), video.ID, errPipelineBoom)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := store.status[video.ID]; got != model.VideoStatusTranscribed {
			t.Errorf("status = %q, want it untouched", got)
		}
	})

	t.Run("StoreErrorIsReturned", func(t *testing.T) {
		video := transcribeFixture()
		store := newPipelineStore(model.VideoStatusScreenedIn, video)
		store.failErr = errPipelineBoom

		err := NewTranscribeService(store, nil, nil).
			MarkTranscribeFailed(t.Context(), video.ID, errors.New("asr down"))

		if !errors.Is(err, errPipelineBoom) {
			t.Fatalf("error = %v, want it to wrap the store error", err)
		}
	})
}

func TestEnqueueScreenedInVideos(t *testing.T) {
	a, b, other := transcribeFixture(), transcribeFixture(), transcribeFixture()
	store := newPipelineStore(model.VideoStatusScreenedIn, a, b, other)
	store.status[other.ID] = model.VideoStatusTranscribed
	transcriber := &pipelineTranscriber{}

	count, err := NewTranscribeService(store, &pipelineMediaResolver{}, transcriber).
		EnqueueScreenedInVideos(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := []uuid.UUID{
		a.ID,
		b.ID,
	}; count != 2 ||
		!slices.Equal(store.transcribeEnqueued, want) {
		t.Errorf("count = %d, transcription queued for %v, want %v",
			count, store.transcribeEnqueued, want)
	}
	if len(store.analyzeEnqueued) != 0 {
		t.Errorf("analysis queued for %v, want none", store.analyzeEnqueued)
	}
	if len(transcriber.gotURLs) != 0 {
		t.Errorf(
			"asr calls = %d, want 0 (transcribing happens in the jobs)",
			len(transcriber.gotURLs),
		)
	}
	if got := store.status[a.ID]; got != model.VideoStatusScreenedIn {
		t.Errorf("status = %q, the sweep must not change it", got)
	}
}

func TestEnqueueScreenedInVideos_ListErrorIsReturned(t *testing.T) {
	store := newPipelineStore(model.VideoStatusScreenedIn)
	store.listErr = errPipelineBoom

	_, err := NewTranscribeService(store, &pipelineMediaResolver{}, &pipelineTranscriber{}).
		EnqueueScreenedInVideos(t.Context())
	if !errors.Is(err, errPipelineBoom) {
		t.Fatalf("error = %v, want it to wrap the list error", err)
	}
}
