package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// screenBatchSize is how many discovered videos are loaded at a time.
const screenBatchSize = 50

// maxFailureReasonLen caps the error text stored with a failed video, because
// analyzer errors can contain the whole raw model output.
const maxFailureReasonLen = 1000

// errNoAnalysis is returned when the analyzer reports success but gives no
// result.
var errNoAnalysis = errors.New("analyzer returned no analysis")

// videoScreenStore is the part of data.VideoRepoer that ScreenService uses.
type videoScreenStore interface {
	ListDiscoveredVideos(
		ctx context.Context,
		limit int32,
		afterCreatedAt time.Time,
		afterID uuid.UUID,
	) ([]*model.Video, error)
	GetVideoForScreening(ctx context.Context, id uuid.UUID) (*model.Video, error)
	SetVideoScreenStatus(
		ctx context.Context,
		id uuid.UUID,
		status model.VideoStatus,
	) (bool, error)
	EnqueueScreenVideoJob(ctx context.Context, videoID uuid.UUID) error
	SetVideoScreenFailed(ctx context.Context, id uuid.UUID, reason string) (bool, error)
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
	var (
		count          int
		afterCreatedAt time.Time
		afterID        uuid.UUID
	)

	for {
		if err := ctx.Err(); err != nil {
			return count, fmt.Errorf("enqueue screening: %w", err)
		}

		videos, err := s.videoRepo.ListDiscoveredVideos(
			ctx, screenBatchSize, afterCreatedAt, afterID,
		)
		if err != nil {
			return count, fmt.Errorf("enqueue screening: %w", err)
		}
		if len(videos) == 0 {
			return count, nil
		}

		for _, v := range videos {
			if err := s.videoRepo.EnqueueScreenVideoJob(ctx, v.ID); err != nil {
				return count, fmt.Errorf("enqueue screening of video %s: %w", v.ID, err)
			}
			count++
		}

		last := videos[len(videos)-1]
		afterCreatedAt, afterID = last.CreatedAt, last.ID
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

// MarkScreenFailed marks a video as failed because screening gave up on it. A
// video that is no longer 'discovered' is left alone.
func (s *ScreenService) MarkScreenFailed(ctx context.Context, id uuid.UUID, cause error) error {
	applied, err := s.videoRepo.SetVideoScreenFailed(ctx, id, failureReason(cause))
	if err != nil {
		return fmt.Errorf("mark video %s as failed: %w", id, err)
	}
	if !applied {
		log.Printf("[ScreenService] video %s is no longer discovered, failure not stored", id)
	}
	return nil
}

// failureReason returns the text of err, cut to maxFailureReasonLen characters.
func failureReason(err error) string {
	runes := []rune(err.Error())
	if len(runes) > maxFailureReasonLen {
		runes = runes[:maxFailureReasonLen]
	}
	return string(runes)
}
