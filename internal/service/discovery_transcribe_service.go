package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// transcribeAudioFormat is the format hint sent to the ASR service for the
// video downloaded by Apify.
const transcribeAudioFormat = "mp4"

// TranscribeService turns the audio of screened-in videos into text.
type TranscribeService struct {
	videoRepo   data.VideoRepoer
	resolver    data.VideoMediaResolver
	transcriber data.SpeechTranscriber
}

// NewTranscribeService constructs a TranscribeService.
func NewTranscribeService(
	videoRepo data.VideoRepoer,
	resolver data.VideoMediaResolver,
	transcriber data.SpeechTranscriber,
) *TranscribeService {
	return &TranscribeService{
		videoRepo:   videoRepo,
		resolver:    resolver,
		transcriber: transcriber,
	}
}

// EnqueueScreenedInVideos enqueues one transcription job for every video that
// is still 'screened_in' and returns how many new jobs it enqueued. A video
// that already has a job in the queue is not queued twice and is not counted.
func (s *TranscribeService) EnqueueScreenedInVideos(ctx context.Context) (int, error) {
	return enqueueVideosByStatus(
		ctx,
		s.videoRepo,
		model.VideoStatusScreenedIn,
		s.videoRepo.EnqueueTranscribeVideoJob,
	)
}

// TranscribeVideo transcribes one video, stores the transcript and queues its
// analysis. The status change and the new job are written in one
// transaction. A video that is gone or no longer 'screened_in' is left alone.
func (s *TranscribeService) TranscribeVideo(ctx context.Context, id uuid.UUID) error {
	video, err := s.videoRepo.GetVideoForTranscription(ctx, id)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			log.Printf("[TranscribeService] video %s is gone or not screened in", id)
			return nil
		}
		return fmt.Errorf("failed to get video %s: %w", id, err)
	}

	mediaURL, err := s.mediaURL(ctx, video)
	if err != nil {
		return err
	}

	transcript, err := s.transcriber.TranscribeURL(ctx, mediaURL, transcribeAudioFormat)
	if err != nil {
		return fmt.Errorf("failed to transcribe video %s: %w", id, err)
	}

	err = s.videoRepo.WithTx(ctx, func(r data.VideoRepoer) error {
		applied, err := r.SetVideoTranscribed(ctx, id, transcript)
		if err != nil {
			return err
		}
		if !applied {
			log.Printf("[TranscribeService] video %s was already transcribed, result dropped", id)
			return nil
		}
		if _, err := r.EnqueueAnalyzeVideoJob(ctx, id); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to store transcript of video %s: %w", id, err)
	}
	return nil
}

// mediaURL returns the stored media URL, or asks Apify for one and stores it.
// Storing it means a retry after a failed transcription does not pay for
// another Apify run.
func (s *TranscribeService) mediaURL(ctx context.Context, video *model.Video) (string, error) {
	if video.MediaURL != nil && *video.MediaURL != "" {
		return *video.MediaURL, nil
	}

	resolved, err := s.resolver.ResolveMediaURL(ctx, video.VideoURL)
	if err != nil {
		return "", fmt.Errorf("failed to resolve media url of video %s: %w", video.ID, err)
	}
	if err := s.videoRepo.SetVideoMediaURL(ctx, video.ID, resolved); err != nil {
		return "", fmt.Errorf("failed to store media url of video %s: %w", video.ID, err)
	}
	return resolved, nil
}

// MarkTranscribeFailed marks a video as failed because transcription gave up
// on it. A video that is no longer 'screened_in' is left alone.
func (s *TranscribeService) MarkTranscribeFailed(
	ctx context.Context,
	id uuid.UUID,
	cause error,
) error {
	applied, err := s.videoRepo.SetVideoFailed(
		ctx,
		id,
		model.VideoStatusScreenedIn,
		model.VideoFailedStageTranscribe,
		failureReason(cause),
	)
	if err != nil {
		return fmt.Errorf("failed to mark video %s as failed: %w", id, err)
	}
	if !applied {
		log.Printf("[TranscribeService] video %s is no longer screened in, failure not stored", id)
	}
	return nil
}
