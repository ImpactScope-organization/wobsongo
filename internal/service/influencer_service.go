package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// tikTokScraper defines the TikTok scraper methods used by this service.
type tikTokScraper interface {
	FetchProfileVideos(
		ctx context.Context,
		username string,
		limit int,
	) ([]dto.TikTokScraperItem, error)
	SearchByKeyword(ctx context.Context, keyword string, limit int) ([]dto.TikTokScraperItem, error)
}

// InfluencerService drives the influencer-monitoring/discovery
// pulling fresh data from TikTok via Apify and persisting it through the repo layer.
type InfluencerService struct {
	influencerRepo      data.InfluencerRepoer
	influencerVideoRepo data.InfluencerVideoRepoer
	scraper             tikTokScraper
}

// NewInfluencerService constructs an InfluencerService.
func NewInfluencerService(
	influencerRepo data.InfluencerRepoer,
	influencerVideoRepo data.InfluencerVideoRepoer,
	scraper tikTokScraper,
) *InfluencerService {
	return &InfluencerService{
		influencerRepo:      influencerRepo,
		influencerVideoRepo: influencerVideoRepo,
		scraper:             scraper,
	}
}

const tiktokPlatform = "tiktok"

// InfluencerMonitorResult summarizes one influencer's check-for-new-videos run.
type InfluencerMonitorResult struct {
	Username          string
	ProfileFound      bool
	TotalVideosSeen   int
	NewVideos         []NewVideoInfo
	AvgUploadInterval *time.Duration
	Error             string
}

// NewVideoInfo describes a video that wasn't tracked before this run.
type NewVideoInfo struct {
	VideoURL  string
	Caption   string
	CreatedAt time.Time
}

// InfluencerDiscoveryResult summarizes one (keyword, author) pair found
// while searching for influencers talking about the same topic.
type InfluencerDiscoveryResult struct {
	Keyword        string
	Username       string
	SourceVideoURL string
	AlreadyKnown   bool
	Error          string
}

// MonitorInfluencers fetches each username's latest videos, stores/updates
// their profile and video metadata, recomputes their average upload
// interval (first tracked video -> last tracked video), and reports what's
// new since the last check.
func (s *InfluencerService) MonitorInfluencers(
	ctx context.Context,
	usernames []string,
	resultsPerProfile int,
) ([]InfluencerMonitorResult, error) {
	results := make([]InfluencerMonitorResult, 0, len(usernames))

	for _, username := range usernames {
		result := InfluencerMonitorResult{Username: username}

		items, err := s.scraper.FetchProfileVideos(ctx, username, resultsPerProfile)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}
		if len(items) == 0 {
			results = append(results, result)
			continue
		}
		result.ProfileFound = true

		influencer := &model.Influencer{
			Platform: tiktokPlatform,
			Username: username,
			Source:   model.InfluencerSourceSeed,
		}
		applyAuthorMeta(influencer, &items[0].AuthorMeta)

		if err := s.influencerRepo.UpsertInfluencer(ctx, influencer); err != nil {
			result.Error = "failed to save profile: " + err.Error()
			results = append(results, result)
			continue
		}

		failedVideos := 0
		for i := range items {
			video := itemToVideo(influencer.ID, &items[i])
			isNew, err := s.influencerVideoRepo.UpsertVideo(ctx, video)
			if err != nil {
				failedVideos++
				continue
			}
			if isNew {
				result.NewVideos = append(result.NewVideos, NewVideoInfo{
					VideoURL:  video.VideoURL,
					Caption:   video.Caption,
					CreatedAt: video.VideoCreatedAt,
				})
			}
		}
		if failedVideos > 0 {
			result.Error = fmt.Sprintf("%d video(s) failed to save", failedVideos)
		}

		allVideos, err := s.influencerVideoRepo.ListByInfluencerID(ctx, influencer.ID)
		if err != nil {
			result.Error = "failed to recompute upload cadence: " + err.Error()
			results = append(results, result)
			continue
		}
		result.TotalVideosSeen = len(allVideos)

		firstAt, lastAt, avgInterval := computeUploadCadence(allVideos)
		result.AvgUploadInterval = avgInterval

		var avgHours *float64
		if avgInterval != nil {
			h := avgInterval.Hours()
			avgHours = &h
		}
		if err := s.influencerRepo.UpdateAggregates(
			ctx, influencer.ID, firstAt, lastAt, avgHours, int64(len(allVideos)),
		); err != nil {
			result.Error = "failed to update aggregates: " + err.Error()
		}

		results = append(results, result)
	}

	return results, nil
}

// DiscoverInfluencers searches TikTok for each keyword and saves any
// author not already tracked as a newly discovered influencer, along with
// the video that surfaced them.
func (s *InfluencerService) DiscoverInfluencers(
	ctx context.Context,
	keywords []string,
	resultsPerKeyword int,
) ([]InfluencerDiscoveryResult, error) {
	tracked, err := s.influencerRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracked influencers: %w", err)
	}

	known := make(map[string]bool, len(tracked))
	for _, inf := range tracked {
		known[inf.Username] = true
	}

	var results []InfluencerDiscoveryResult

	for _, keyword := range keywords {
		items, err := s.scraper.SearchByKeyword(ctx, keyword, resultsPerKeyword)
		if err != nil {
			results = append(results, InfluencerDiscoveryResult{
				Keyword: keyword,
				Error:   err.Error(),
			})
			continue
		}

		for i := range items {
			item := &items[i]
			username := item.AuthorMeta.Name
			if username == "" {
				continue
			}

			result := InfluencerDiscoveryResult{
				Keyword:        keyword,
				Username:       username,
				SourceVideoURL: item.WebVideoURL,
			}

			if known[username] {
				result.AlreadyKnown = true
				results = append(results, result)
				continue
			}
			known[username] = true

			influencer := &model.Influencer{
				Platform:               tiktokPlatform,
				Username:               username,
				Source:                 model.InfluencerSourceDiscovered,
				DiscoveredViaKeyword:   keyword,
				DiscoveredFromVideoURL: item.WebVideoURL,
			}
			applyAuthorMeta(influencer, &item.AuthorMeta)

			if err := s.influencerRepo.UpsertInfluencer(ctx, influencer); err != nil {
				results = append(results, result)
				continue
			}

			if _, err := s.influencerVideoRepo.UpsertVideo(
				ctx,
				itemToVideo(influencer.ID, item),
			); err != nil {
				_ = err
			}

			results = append(results, result)
		}
	}

	return results, nil
}

// computeUploadCadence returns the oldest and newest video timestamps
// and the average gap between tracked uploads.
func computeUploadCadence(
	videos []*model.InfluencerVideo,
) (firstAt, lastAt *time.Time, avgInterval *time.Duration) {
	if len(videos) == 0 {
		return nil, nil, nil
	}

	first := videos[0].VideoCreatedAt
	last := videos[len(videos)-1].VideoCreatedAt
	firstAt = &first
	lastAt = &last

	if len(videos) < 2 || !last.After(first) {
		return firstAt, lastAt, nil
	}

	avg := last.Sub(first) / time.Duration(len(videos)-1)
	avgInterval = &avg
	return firstAt, lastAt, avgInterval
}

func applyAuthorMeta(influencer *model.Influencer, author *dto.TikTokAuthorMeta) {
	if author == nil {
		return
	}
	influencer.ProfileURL = "https://www.tiktok.com/@" + author.Name
	influencer.Nickname = author.NickName
	influencer.Bio = author.Signature
	influencer.AvatarURL = author.Avatar
	influencer.Verified = author.Verified
	influencer.FollowersCount = author.Fans
	influencer.FollowingCount = author.Following
	influencer.HeartsCount = author.Heart
	influencer.ReportedVideoCount = author.Video
}

func itemToVideo(influencerID uuid.UUID, item *dto.TikTokScraperItem) *model.InfluencerVideo {
	hashtags := make([]string, 0, len(item.Hashtags))
	for _, h := range item.Hashtags {
		if h.Name != "" {
			hashtags = append(hashtags, h.Name)
		}
	}

	return &model.InfluencerVideo{
		InfluencerID:   influencerID,
		TikTokVideoID:  item.ID,
		VideoURL:       item.WebVideoURL,
		Caption:        item.Text,
		LikeCount:      item.DiggCount,
		CommentCount:   item.CommentCount,
		ShareCount:     item.ShareCount,
		PlayCount:      item.PlayCount,
		CollectCount:   item.CollectCount,
		Hashtags:       hashtags,
		VideoCreatedAt: item.CreateTimeISO,
	}
}
