package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// screenBatchSize is how many discovered videos are loaded at a time.
const screenBatchSize = 50

// videoScreenStore is the part of data.VideoRepoer that ScreenService uses.
type videoScreenStore interface {
	ListDiscoveredVideos(
		ctx context.Context,
		limit int32,
		excludeIDs []uuid.UUID,
	) ([]*model.Video, error)
	SetVideoScreenStatus(
		ctx context.Context,
		id uuid.UUID,
		status model.VideoStatus,
	) (bool, error)
}

// ScreenService runs the first screening of discovered videos: it decides
// from the caption and hashtags whether a video is worth transcribing.
type ScreenService struct {
	videoRepo   videoScreenStore
	analyzer    data.ClaimAnalyzer
	concurrency int
}

// NewScreenService constructs a ScreenService. concurrency is how many videos
// are analyzed at the same time; a value below 1 is treated as 1.
func NewScreenService(
	videoRepo videoScreenStore,
	analyzer data.ClaimAnalyzer,
	concurrency int,
) *ScreenService {
	return &ScreenService{
		videoRepo:   videoRepo,
		analyzer:    analyzer,
		concurrency: max(concurrency, 1),
	}
}

// ScreenSummary counts the outcome of one screening run.
type ScreenSummary struct {
	ScreenedIn  int
	ScreenedOut int
	Failed      int
}

// screenOutcome is the result of screening one video.
type screenOutcome struct {
	status model.VideoStatus
	err    error
}

// ScreenDiscoveredVideos screens every video that is still 'discovered',
// batch by batch, until none is left.
func (s *ScreenService) ScreenDiscoveredVideos(ctx context.Context) (ScreenSummary, error) {
	var summary ScreenSummary
	skip := make([]uuid.UUID, 0)

	for {
		if err := ctx.Err(); err != nil {
			return summary, err
		}

		videos, err := s.videoRepo.ListDiscoveredVideos(ctx, screenBatchSize, skip)
		if err != nil {
			return summary, fmt.Errorf("failed to list discovered videos: %w", err)
		}
		if len(videos) == 0 {
			return summary, nil
		}

		outcomes := s.screenBatch(ctx, videos)

		var lastErr error
		failed := 0
		for i, outcome := range outcomes {
			if outcome.err != nil {
				failed++
				lastErr = outcome.err
				skip = append(skip, videos[i].ID)
				log.Printf("[ScreenService] %v", outcome.err)
				continue
			}
			if outcome.status == model.VideoStatusScreenedIn {
				summary.ScreenedIn++
			} else {
				summary.ScreenedOut++
			}
		}
		summary.Failed += failed

		if failed == len(videos) {
			return summary, fmt.Errorf("every video of a batch failed to screen: %w", lastErr)
		}
	}
}

// screenBatch screens the videos with a bounded number of analyses at once.
// The outcome of videos[i] is at index i of the result.
func (s *ScreenService) screenBatch(ctx context.Context, videos []*model.Video) []screenOutcome {
	outcomes := make([]screenOutcome, len(videos))
	sem := make(chan struct{}, s.concurrency)

	var wg sync.WaitGroup
	for i, v := range videos {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			status, err := s.screenVideo(ctx, v)
			outcomes[i] = screenOutcome{status: status, err: err}
		}()
	}
	wg.Wait()
	return outcomes
}

// screenVideo decides and stores the screening status of one video.
func (s *ScreenService) screenVideo(
	ctx context.Context,
	v *model.Video,
) (model.VideoStatus, error) {
	status := model.VideoStatusScreenedOut

	if text := screenText(v); text != "" {
		analysis, err := s.analyzer.Analyze(ctx, text, nil)
		if err != nil {
			return "", fmt.Errorf("failed to analyze video %s: %w", v.ID, err)
		}
		if analysis.InScope {
			status = model.VideoStatusScreenedIn
		}
	}

	applied, err := s.videoRepo.SetVideoScreenStatus(ctx, v.ID, status)
	if err != nil {
		return "", fmt.Errorf("failed to store screen status of video %s: %w", v.ID, err)
	}
	if !applied {
		log.Printf("[ScreenService] video %s was already screened, result dropped", v.ID)
	}
	return status, nil
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
