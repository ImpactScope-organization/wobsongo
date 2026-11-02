package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

const (
	// crawlStaleAfter is how long a run may stay 'running' before it is
	// considered dead (for example after a server crash).
	crawlStaleAfter = 2 * time.Hour

	// maxRunErrorLength caps the error text stored on a crawl run.
	maxRunErrorLength = 2000

	// monitorMaxCreatorsPerRun limits the number of influencers monitored per run.
	// Will be configurable via an environment variable in a follow-up PR.
	monitorMaxCreatorsPerRun = 25

	// monitorDefaultInterval is the wait before the next check of a creator
	// whose upload gap cannot be computed yet (fewer than two videos).
	monitorDefaultInterval = 24 * time.Hour

	// monitorMaxInterval is the longest wait before the next check of a
	// creator, so a creator who rarely posts is still checked at least once
	// in this period.
	// Will be configurable via an environment variable in a follow-up PR.
	monitorMaxInterval = 7 * 24 * time.Hour

	// monitorRetryDelay is how long to wait before checking a creator again
	// after a failed or empty fetch.
	monitorRetryDelay = 24 * time.Hour

	// monitorMaxEmptyChecks is how many checks in a row may find no usable
	// video before the creator is deactivated.
	monitorMaxEmptyChecks = 3

	// dueTolerance absorbs the small delay between a periodic tick and the
	// moment a run records its started_at. Without it, the next tick can
	// land a few seconds before last+interval and the run is skipped until
	// the tick after that (a 24h interval silently becomes 48h).
	dueTolerance = 5 * time.Minute
)

// CrawlService runs the discovery (by keyword) and monitoring (by creator)
// steps of the video pipeline and records every run in crawl_runs.
type CrawlService struct {
	crawlerRepo    data.CrawlerRepoer
	influencerRepo data.InfluencerRepoer
	videoRepo      data.VideoRepoer
	scraper        tikTokScraper
	now            func() time.Time
}

// NewCrawlService constructs a CrawlService.
func NewCrawlService(
	crawlerRepo data.CrawlerRepoer,
	influencerRepo data.InfluencerRepoer,
	videoRepo data.VideoRepoer,
	scraper tikTokScraper,
) *CrawlService {
	return &CrawlService{
		crawlerRepo:    crawlerRepo,
		influencerRepo: influencerRepo,
		videoRepo:      videoRepo,
		scraper:        scraper,
		now:            time.Now,
	}
}

type discoverRunParams struct {
	Keywords   []string `json:"keywords"`
	MaxResults int      `json:"max_results"`
}

type monitorRunParams struct {
	Influencers []string `json:"influencers"`
	MaxResults  int      `json:"max_results"`
}

// RunDiscovery searches every active keyword and stores the creators and
// videos it finds. A scheduled run is skipped until the discovery interval
// has passed since the last run; a manual run always goes ahead.
func (s *CrawlService) RunDiscovery(
	ctx context.Context,
	trigger model.CrawlTrigger,
	triggeredBy *uuid.UUID,
) error {
	setting, err := s.crawlerRepo.GetSetting(ctx)
	if err != nil {
		return fmt.Errorf("failed to load crawler setting: %w", err)
	}

	if trigger == model.CrawlTriggerScheduled {
		interval := time.Duration(setting.DiscoveryIntervalHours) * time.Hour
		due, err := s.isDue(ctx, model.CrawlKindDiscover, interval)
		if err != nil {
			return err
		}
		if !due {
			return nil
		}
	}

	keywords, err := s.crawlerRepo.ListActiveKeywords(ctx)
	if err != nil {
		return fmt.Errorf("failed to list keywords: %w", err)
	}
	if len(keywords) == 0 {
		log.Printf("[CrawlService] discovery skipped: no active keyword")
		return nil
	}

	names := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		names = append(names, kw.Keyword)
	}

	run, err := s.startRun(ctx, model.CrawlKindDiscover, trigger, triggeredBy,
		discoverRunParams{Keywords: names, MaxResults: setting.MaxResultsKeyword})
	if errors.Is(err, data.ErrConflict) {
		log.Printf("[CrawlService] discovery skipped: a run is already in progress")
		return nil
	}
	if err != nil {
		return err
	}

	var failures []string
	authors := make(map[string]uuid.UUID)

	for _, kw := range keywords {
		if err := ctx.Err(); err != nil {
			failures = append(failures, "interrupted: "+err.Error())
			break
		}

		found, created, err := s.discoverKeyword(ctx, kw, setting.MaxResultsKeyword, authors)
		run.VideoFound += found
		run.VideoNew += created
		if err != nil {
			failures = append(failures, fmt.Sprintf("%q: %v", kw.Keyword, err))
			continue
		}

		if err := s.crawlerRepo.MarkKeywordRun(ctx, kw.ID); err != nil {
			log.Printf("[CrawlService] failed to mark keyword %q: %v", kw.Keyword, err)
		}
	}

	return s.finishRun(ctx, run, len(keywords), failures)
}

// RunMonitor checks creators for new videos. With influencerID set it checks
// only that creator (admin "Check Now"); otherwise it checks the creators
// whose next check is due.
func (s *CrawlService) RunMonitor(
	ctx context.Context,
	trigger model.CrawlTrigger,
	triggeredBy *uuid.UUID,
	influencerID *uuid.UUID,
) error {
	setting, err := s.crawlerRepo.GetSetting(ctx)
	if err != nil {
		return fmt.Errorf("failed to load crawler setting: %w", err)
	}

	var creators []*model.Influencer
	if influencerID != nil {
		creator, err := s.influencerRepo.GetByID(ctx, *influencerID)
		if err != nil {
			return fmt.Errorf("failed to load influencer: %w", err)
		}
		creators = []*model.Influencer{creator}
	} else {
		creators, err = s.influencerRepo.ListDue(ctx, monitorMaxCreatorsPerRun)
		if err != nil {
			return fmt.Errorf("failed to list due influencers: %w", err)
		}
	}
	if len(creators) == 0 {
		return nil
	}

	usernames := make([]string, 0, len(creators))
	for _, c := range creators {
		usernames = append(usernames, c.Username)
	}

	run, err := s.startRun(ctx, model.CrawlKindMonitor, trigger, triggeredBy,
		monitorRunParams{Influencers: usernames, MaxResults: setting.MaxResultsInfluencer})
	if errors.Is(err, data.ErrConflict) {
		log.Printf("[CrawlService] monitor skipped: a run is already in progress")
		return nil
	}
	if err != nil {
		return err
	}

	var failures []string

	for _, creator := range creators {
		if err := ctx.Err(); err != nil {
			failures = append(failures, "interrupted: "+err.Error())
			break
		}

		found, created, err := s.monitorCreator(ctx, creator, setting.MaxResultsInfluencer)
		run.VideoFound += found
		run.VideoNew += created
		if err != nil {
			failures = append(failures, fmt.Sprintf("@%s: %v", creator.Username, err))
		}
	}

	return s.finishRun(ctx, run, len(creators), failures)
}

// discoverKeyword searches one keyword and stores each author and video.
func (s *CrawlService) discoverKeyword(
	ctx context.Context,
	kw *model.DiscoveryKeyword,
	limit int,
	authors map[string]uuid.UUID,
) (found, created int, err error) {
	items, err := s.scraper.SearchByKeyword(ctx, kw.Keyword, limit)
	if err != nil {
		return 0, 0, fmt.Errorf("apify search failed: %w", err)
	}

	for i := range items {
		item := &items[i]
		if !isUsableItem(item) || item.AuthorMeta.Name == "" {
			continue
		}
		found++

		influencerID := s.upsertAuthor(ctx, item, kw.Keyword, authors)
		video := itemToDiscoveredVideo(
			item, item.AuthorMeta.Name, influencerID, model.VideoOriginDiscovery, kw.Keyword,
		)
		if s.saveVideo(ctx, video) {
			created++
		}
	}
	return found, created, nil
}

// monitorCreator fetches one creator's latest videos, stores them and
// schedules the creator's next check from their upload cadence.
func (s *CrawlService) monitorCreator(
	ctx context.Context,
	creator *model.Influencer,
	limit int,
) (found, created int, err error) {
	items, err := s.scraper.FetchProfileVideos(ctx, creator.Username, limit)
	if err != nil {
		s.scheduleNextCheck(ctx, creator.ID, monitorRetryDelay)
		return 0, 0, fmt.Errorf("apify profile fetch failed: %w", err)
	}

	usable := make([]*dto.TikTokScraperItem, 0, len(items))
	for i := range items {
		if isUsableItem(&items[i]) {
			usable = append(usable, &items[i])
		}
	}
	if len(usable) == 0 {
		if err := s.handleEmptyCheck(ctx, creator); err != nil {
			return 0, 0, fmt.Errorf("failed to record empty check: %w", err)
		}
		return 0, 0, nil
	}

	profile := *creator
	applyAuthorMeta(&profile, &usable[0].AuthorMeta)
	if err := s.influencerRepo.UpsertInfluencer(ctx, &profile); err != nil {
		log.Printf("[CrawlService] failed to refresh profile @%s: %v", creator.Username, err)
	}

	creatorID := creator.ID
	for _, item := range usable {
		found++

		video := itemToDiscoveredVideo(
			item, creator.Username, &creatorID, model.VideoOriginMonitor, "",
		)
		if s.saveVideo(ctx, video) {
			created++
		}
	}

	s.updateCadence(ctx, creator)
	return found, created, nil
}

// handleEmptyCheck deals with a check that found no usable video: it counts
// the miss and either retries later or, after monitorMaxEmptyChecks in a row,
// stops monitoring the creator.
func (s *CrawlService) handleEmptyCheck(
	ctx context.Context,
	creator *model.Influencer,
) error {
	nextCheckAt := s.now().Add(monitorRetryDelay)
	stillActive, err := s.influencerRepo.RecordEmptyCheck(
		ctx, creator.ID, monitorMaxEmptyChecks, nextCheckAt,
	)
	if err != nil {
		return err
	}

	if !stillActive {
		log.Printf(
			"[CrawlService] @%s deactivated: no usable video in %d checks in a row",
			creator.Username, monitorMaxEmptyChecks,
		)
	}
	return nil
}

// updateCadence recomputes the creator's upload cadence from every stored
// video and schedules the next check.
func (s *CrawlService) updateCadence(ctx context.Context, creator *model.Influencer) {
	times, err := s.videoRepo.ListVideoTimesByInfluencerID(ctx, creator.ID)
	if err != nil {
		log.Printf("[CrawlService] failed to list videos of @%s: %v", creator.Username, err)
		s.scheduleNextCheck(ctx, creator.ID, monitorDefaultInterval)
		return
	}

	firstAt, lastAt, avg := cadenceFromTimes(times)

	var avgHours *float64
	if avg != nil {
		h := avg.Hours()
		avgHours = &h
	}
	err = s.influencerRepo.UpdateAggregates(
		ctx, creator.ID, firstAt, lastAt, avgHours, int64(len(times)),
	)
	if err != nil {
		log.Printf("[CrawlService] failed to update aggregates of @%s: %v", creator.Username, err)
	}

	s.scheduleNextCheck(ctx, creator.ID, nextCheckInterval(avg))
}

// scheduleNextCheck sets the influencer's next check time.
func (s *CrawlService) scheduleNextCheck(ctx context.Context, id uuid.UUID, after time.Duration) {
	if err := s.influencerRepo.SetNextCheckAt(ctx, id, s.now().Add(after)); err != nil {
		log.Printf("[CrawlService] failed to schedule next check of %s: %v", id, err)
	}
}

// upsertAuthor stores the author of a discovered video once per run and
// returns their ID. It returns nil if the author could not be saved; the
// video is still stored, just without a creator link.
func (s *CrawlService) upsertAuthor(
	ctx context.Context,
	item *dto.TikTokScraperItem,
	keyword string,
	cache map[string]uuid.UUID,
) *uuid.UUID {
	name := item.AuthorMeta.Name
	if id, ok := cache[name]; ok {
		return &id
	}

	influencer := &model.Influencer{
		Platform:               tiktokPlatform,
		Username:               name,
		Source:                 model.InfluencerSourceDiscovered,
		DiscoveredViaKeyword:   keyword,
		DiscoveredFromVideoURL: item.WebVideoURL,
	}
	applyAuthorMeta(influencer, &item.AuthorMeta)

	if err := s.influencerRepo.UpsertInfluencer(ctx, influencer); err != nil {
		log.Printf("[CrawlService] failed to save author @%s: %v", name, err)
		return nil
	}

	cache[name] = influencer.ID
	id := influencer.ID
	return &id
}

// saveVideo stores one video and reports whether it was new. A conflict with
// another unique key (for example the same URL under a different external
// ID) means the video is already known, so it is skipped quietly.
func (s *CrawlService) saveVideo(ctx context.Context, v *model.Video) bool {
	isNew, err := s.videoRepo.UpsertDiscoveredVideo(ctx, v)
	if err != nil {
		if !errors.Is(err, data.ErrConflict) {
			log.Printf("[CrawlService] failed to save video %s: %v", v.VideoURL, err)
		}
		return false
	}
	if isNew {
		s.onNewVideo(ctx, v)
	}
	return isNew
}

// onNewVideo runs for every video stored for the first time.
func (s *CrawlService) onNewVideo(_ context.Context, _ *model.Video) {
	// TODO(screen): enqueue ScreenVideoJob{VideoID: v.ID} once step 3 exists.
}

// isDue checks whether enough time has passed since the last run,
// allowing a small delay in the scheduler's execution time.
func (s *CrawlService) isDue(
	ctx context.Context,
	kind model.CrawlKind,
	interval time.Duration,
) (bool, error) {
	last, err := s.crawlerRepo.GetLastRunStartedAt(ctx, kind)
	if err != nil {
		return false, fmt.Errorf("failed to check last %s run: %w", kind, err)
	}
	if last == nil {
		return true, nil
	}

	tolerance := min(dueTolerance, interval/2)
	return !s.now().Before(last.Add(interval - tolerance)), nil
}

// startRun fails runs that look dead, then records a new running run. It
// returns data.ErrConflict if a run of the same kind is already running.
func (s *CrawlService) startRun(
	ctx context.Context,
	kind model.CrawlKind,
	trigger model.CrawlTrigger,
	triggeredBy *uuid.UUID,
	params any,
) (*model.CrawlRun, error) {
	payload, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run params: %w", err)
	}

	if err := s.crawlerRepo.FailStaleRuns(ctx, kind, crawlStaleAfter); err != nil {
		return nil, err
	}

	run := &model.CrawlRun{
		Kind:        kind,
		Trigger:     trigger,
		TriggeredBy: triggeredBy,
		Params:      payload,
	}
	if err := s.crawlerRepo.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// finishRun saves the outcome. The run fails only when every item failed;
// otherwise it succeeds and keeps the partial failures in its error field.
func (s *CrawlService) finishRun(
	ctx context.Context,
	run *model.CrawlRun,
	total int,
	failures []string,
) error {
	run.Status = model.CrawlStatusSucceeded

	var runErr error
	if len(failures) > 0 {
		msg := strings.Join(failures, "; ")
		msg = truncateUTF8(msg, maxRunErrorLength)
		run.Error = &msg

		if len(failures) >= total {
			run.Status = model.CrawlStatusFailed
			runErr = fmt.Errorf("%s run %s failed: %s", run.Kind, run.ID, msg)
		}
	}

	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	if err := s.crawlerRepo.FinishRun(saveCtx, run); err != nil {
		return fmt.Errorf("failed to finish crawl run %s: %w", run.ID, err)
	}

	log.Printf(
		"[CrawlService] %s run %s %s: found=%d new=%d",
		run.Kind, run.ID, run.Status, run.VideoFound, run.VideoNew,
	)
	return runErr
}

func isUsableItem(item *dto.TikTokScraperItem) bool {
	return item.ID != "" && item.WebVideoURL != ""
}

// itemToDiscoveredVideo converts a scraper item to a video row.
func itemToDiscoveredVideo(
	item *dto.TikTokScraperItem,
	authorName string,
	influencerID *uuid.UUID,
	origin model.VideoOrigin,
	keyword string,
) *model.Video {
	hashtags := make([]string, 0, len(item.Hashtags))
	for _, h := range item.Hashtags {
		if h.Name != "" {
			hashtags = append(hashtags, h.Name)
		}
	}

	externalID := item.ID
	video := &model.Video{
		InfluencerID:     influencerID,
		ExternalID:       &externalID,
		VideoURL:         item.WebVideoURL,
		AuthorUsername:   authorName,
		AuthorProfileURL: "https://www.tiktok.com/@" + authorName,
		Caption:          item.Text,
		Hashtags:         hashtags,
		PlayCount:        item.PlayCount,
		LikeCount:        item.DiggCount,
		CommentCount:     item.CommentCount,
		ShareCount:       item.ShareCount,
		CollectCount:     item.CollectCount,
		VideoCreatedAt:   item.CreateTimeISO,
		VideoType:        tiktokPlatform,
		Origin:           origin,
	}
	if keyword != "" {
		video.DiscoveredViaKeyword = &keyword
	}
	return video
}

// cadenceFromTimes returns the oldest and newest publish times and the
// average gap between uploads.
func cadenceFromTimes(times []time.Time) (firstAt, lastAt *time.Time, avg *time.Duration) {
	if len(times) == 0 {
		return nil, nil, nil
	}

	first, last := times[0], times[len(times)-1]
	firstAt, lastAt = &first, &last

	if len(times) < 2 || !last.After(first) {
		return firstAt, lastAt, nil
	}

	gap := last.Sub(first) / time.Duration(len(times)-1)
	return firstAt, lastAt, &gap
}

// nextCheckInterval returns the wait before the next check: the creator's
// average upload gap (at most monitorMaxInterval), or the default when the
// gap is not known yet.
func nextCheckInterval(avg *time.Duration) time.Duration {
	if avg == nil {
		return monitorDefaultInterval
	}
	return min(*avg, monitorMaxInterval)
}

// truncateUTF8 shortens s to at most maxBytes without cutting a multi-byte
// character in half, which Postgres would reject as invalid UTF-8.
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
