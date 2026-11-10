package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// screenBatchSize is how many discovered videos are loaded at a time.
const screenBatchSize = 50

// errNoAnalysis is returned when the analyzer reports success but gives no
// result.
var errNoAnalysis = errors.New("analyzer returned no analysis")

// videoScreenStore is the part of data.VideoRepoer that ScreenService uses.
type videoScreenStore interface {
	ListDiscoveredVideos(
		ctx context.Context,
		limit int32,
		excludeIDs []uuid.UUID,
	) ([]*model.Video, error)
	GetVideoForScreening(ctx context.Context, id uuid.UUID) (*model.Video, error)
	SetVideoScreenStatus(
		ctx context.Context,
		id uuid.UUID,
		status model.VideoStatus,
	) (bool, error)
	EnqueueScreenVideoJob(ctx context.Context, videoID uuid.UUID) error
}

// ScreenService runs the first screening of discovered videos: it decides
// from the caption and hashtags whether a video is worth transcribing.
type ScreenService struct {
	videoRepo videoScreenStore
	analyzer  data.ClaimAnalyzer
}

// NewScreenService constructs a ScreenService.
func NewScreenService(videoRepo videoScreenStore, analyzer data.ClaimAnalyzer) *ScreenService {
	return &ScreenService{
		videoRepo: videoRepo,
		analyzer:  analyzer,
	}
}

// EnqueueDiscoveredVideos enqueues one screening job for every video that is
// still 'discovered', batch by batch, and returns how many it enqueued. A
// video that already has a job in the queue is not queued twice.
func (s *ScreenService) EnqueueDiscoveredVideos(ctx context.Context) (int, error) {
	enqueued := make([]uuid.UUID, 0)

	for {
		if err := ctx.Err(); err != nil {
			return len(enqueued), fmt.Errorf("enqueue screening: %w", err)
		}

		videos, err := s.videoRepo.ListDiscoveredVideos(ctx, screenBatchSize, enqueued)
		if err != nil {
			return len(enqueued), fmt.Errorf("enqueue screening: %w", err)
		}
		if len(videos) == 0 {
			return len(enqueued), nil
		}

		for _, v := range videos {
			if err := s.videoRepo.EnqueueScreenVideoJob(ctx, v.ID); err != nil {
				return len(enqueued), fmt.Errorf("enqueue screening of video %s: %w", v.ID, err)
			}
			enqueued = append(enqueued, v.ID)
		}
	}
}

// ScreenVideo decides and stores the screening status of one video. A video
// that is gone or no longer 'discovered' is left alone.
func (s *ScreenService) ScreenVideo(ctx context.Context, id uuid.UUID) error {
	video, err := s.videoRepo.GetVideoForScreening(ctx, id)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			log.Printf("[ScreenService] video %s is gone or already screened", id)
			return nil
		}
		return fmt.Errorf("get video %s: %w", id, err)
	}

	status := model.VideoStatusScreenedOut

	text := screenText(video)
	if text != "" {
		analysis, err := s.analyzer.Analyze(ctx, text, nil)
		if err != nil {
			return fmt.Errorf("analyze video %s: %w", id, err)
		}
		if analysis == nil {
			return fmt.Errorf("analyze video %s: %w", id, errNoAnalysis)
		}
		if analysis.InScope {
			status = model.VideoStatusScreenedIn
		}
	}

	applied, err := s.videoRepo.SetVideoScreenStatus(ctx, id, status)
	if err != nil {
		return fmt.Errorf("store screen status of video %s: %w", id, err)
	}
	if !applied {
		log.Printf("[ScreenService] video %s was already screened, result dropped", id)
	}
	return nil
}

// screenText builds the text to analyze: the caption as it is, followed by the
// hashtags. Nothing is trimmed, filtered or cut. It is empty only when both
// the caption and the hashtags are blank.
func screenText(v *model.Video) string {
	var b strings.Builder
	b.WriteString(v.Caption)
	for _, h := range v.Hashtags {
		b.WriteString(" #")
		b.WriteString(h)
	}

	text := b.String()
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return text
}
