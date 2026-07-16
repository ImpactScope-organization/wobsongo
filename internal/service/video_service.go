package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5/pgtype"
)

type VideoService struct {
	videoRepo data.VideoRepoer
}

func NewVideoService(
	videoRepo data.VideoRepoer,
) *VideoService {
	return &VideoService{
		videoRepo: videoRepo,
	}
}

// ProcessAndSaveApifyItems processes a list of Apify TikTok items and saves their metadata to the database.
func (s *VideoService) ProcessAndSaveApifyItems(
	ctx context.Context,
	items []dto.ApifyTikTokItem,
	extractionID string,
) error {
	var errs []error

	for i := range items {
		item := &items[i]

		extractedHashtags := []string{}
		for _, ht := range item.Hashtags {
			if ht.Name != "" {
				extractedHashtags = append(extractedHashtags, ht.Name)
			}
		}

		videoData := &model.Video{
			VideoURL:         item.SubmittedVideoURL,
			AuthorUsername:   item.AuthorMetadata.Name,
			AuthorProfileURL: item.AuthorMetadata.ProfileURL,
			Caption:          item.Text,
			PlayCount:        item.PlayCount,
			LikeCount:        item.DiggCount,
			ThumbnailURL:     item.VideoMetadata.CoverUrl,
			LocationCreated:  item.LocationCreated,
			VideoCreatedAt:   item.CreateTimeISO,
			VideoType:        "tiktok",
			Hashtags:         extractedHashtags,
		}

		// Wrap the database insert and River queue operations in a single transaction.
		err := s.videoRepo.WithTx(ctx, func(txRepo data.VideoRepoer) error {
			// Save the video metadata to the videos table.
			if err := txRepo.CreateVideos(ctx, videoData); err != nil {
				return fmt.Errorf("failed to save video: %w", err)
			}

			// Enqueue a transcription job if a media download URL is available.
			if len(item.MediaUrls) > 0 {
				payload := queue.TranscriptionJob{
					ExtractionID: extractionID,
					VideoID:      videoData.ID,
					DownloadURL:  item.MediaUrls[0],
				}
				if err := txRepo.EnqueueTranscriptionJob(ctx, payload); err != nil {
					return fmt.Errorf("failed to enqueue transcription job: %w", err)
				}
			}

			return nil
		})
		if err != nil {
			log.Printf("Transaction failed for video %s: %v", item.SubmittedVideoURL, err)
			errs = append(errs, fmt.Errorf("video %s: %w", item.SubmittedVideoURL, err))
			continue
		}

		log.Printf(
			"Successfully saved metadata and enqueued job to DB. UUID: %s | URL: %s",
			videoData.ID,
			videoData.VideoURL,
		)
	}

	return errors.Join(errs...)
}

// UpdateVideoTranscription updates the transcription text for a video by its ID.
func (s *VideoService) UpdateVideoTranscription(
	ctx context.Context,
	text pgtype.Text,
	id uuid.UUID,
) error {
	return s.videoRepo.UpdateVideoTranscription(ctx, text, id)
}
