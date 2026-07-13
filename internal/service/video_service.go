package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
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
) error {
	var errs []error

	for i := range items {
		item := &items[i]

		videoData := &model.Video{
			VideoURL:         item.SubmittedVideoURL,
			AuthorUsername:   item.AuthorMeta.Name,
			AuthorProfileURL: item.AuthorMeta.ProfileURL,
			Caption:          item.Text,
			PlayCount:        item.PlayCount,
			LikeCount:        item.DiggCount,
			ThumbnailURL:     item.VideoMeta.CoverUrl,
			LocationCreated:  item.LocationCreated,
			VideoCreatedAt:   item.CreateTimeISO,
			VideoType:        "tiktok",
		}

		if err := s.videoRepo.CreateVideos(ctx, videoData); err != nil {
			log.Printf("Failed to save video %s: %v", item.SubmittedVideoURL, err)
			errs = append(errs, fmt.Errorf("video %s: %w", item.SubmittedVideoURL, err))
			continue
		}

		log.Printf(
			"Successfully saved metadata to DB. UUID: %s | URL: %s",
			videoData.ID,
			videoData.VideoURL,
		)
	}

	return errors.Join(errs...)
}
