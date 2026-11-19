package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// ClaimChecker checks the claims of a text against the knowledge base.
// *ClaimService implements it.
type ClaimChecker interface {
	CheckClaim(ctx context.Context, req *dto.CheckClaimDTO) (*ClaimCheckResult, error)
}

// AnalyzeService extracts the claims of a transcribed video, verifies them
// against the knowledge base and stores them. ClaimService also queues the
// narrative clustering of every stored claim.
type AnalyzeService struct {
	videoRepo data.VideoRepoer
	claims    ClaimChecker
}

// NewAnalyzeService constructs an AnalyzeService. claimService must have the
// narrative flow turned on (WithNarratives), otherwise claims are not stored.
func NewAnalyzeService(videoRepo data.VideoRepoer, claims ClaimChecker) *AnalyzeService {
	return &AnalyzeService{videoRepo: videoRepo, claims: claims}
}

// EnqueueTranscribedVideos enqueues one analysis job for every video that is
// still 'transcribed' and returns how many new jobs it enqueued. A video that
// already has a job in the queue is not queued twice and is not counted.
func (s *AnalyzeService) EnqueueTranscribedVideos(ctx context.Context) (int, error) {
	return enqueueVideosByStatus(
		ctx,
		s.videoRepo,
		model.VideoStatusTranscribed,
		s.videoRepo.EnqueueAnalyzeVideoJob,
	)
}

// AnalyzeVideo analyzes one video and stores its final status: 'verified', or
// 'out_of_scope' when there is nothing to check. A video that is gone or no
// longer 'transcribed' is left alone.
func (s *AnalyzeService) AnalyzeVideo(ctx context.Context, id uuid.UUID) error {
	video, err := s.videoRepo.GetVideoForAnalysis(ctx, id)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			log.Printf("[AnalyzeService] video %s is gone or not transcribed", id)
			return nil
		}
		return fmt.Errorf("failed to get video %s: %w", id, err)
	}

	status, err := s.analyze(ctx, video)
	if err != nil {
		return err
	}

	applied, err := s.videoRepo.SetVideoAnalyzed(ctx, id, status)
	if err != nil {
		return fmt.Errorf("failed to store analysis status of video %s: %w", id, err)
	}
	if !applied {
		log.Printf("[AnalyzeService] video %s was already analyzed, result dropped", id)
	}
	return nil
}

// analyze runs the claim check and returns the status the video ends in.
func (s *AnalyzeService) analyze(
	ctx context.Context,
	video *model.Video,
) (model.VideoStatus, error) {
	if video.TranscriptionText == nil || strings.TrimSpace(*video.TranscriptionText) == "" {
		return model.VideoStatusOutOfScope, nil
	}

	videoID := video.ID
	result, err := s.claims.CheckClaim(ctx, &dto.CheckClaimDTO{
		Text: *video.TranscriptionText,
		Source: &dto.ClaimSource{
			Platform:   model.ClaimPlatformTikTok,
			Format:     string(model.ClaimFormatVideo),
			VideoID:    &videoID,
			OccurredAt: video.VideoCreatedAt,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to check claims of video %s: %w", video.ID, err)
	}
	if !result.InScope {
		return model.VideoStatusOutOfScope, nil
	}
	return model.VideoStatusVerified, nil
}

// MarkAnalyzeFailed marks a video as failed because analysis gave up on it. A
// video that is no longer 'transcribed' is left alone.
func (s *AnalyzeService) MarkAnalyzeFailed(ctx context.Context, id uuid.UUID, cause error) error {
	applied, err := s.videoRepo.SetVideoFailed(
		ctx, id, model.VideoStatusTranscribed, model.VideoFailedStageAnalyze, failureReason(cause),
	)
	if err != nil {
		return fmt.Errorf("failed to mark video %s as failed: %w", id, err)
	}
	if !applied {
		log.Printf("[AnalyzeService] video %s is no longer transcribed, failure not stored", id)
	}
	return nil
}
