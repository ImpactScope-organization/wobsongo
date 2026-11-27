package repo_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// screenTestEpoch is far in the past, so the videos these tests insert are the
// oldest rows of the table whatever else the local database holds.
var screenTestEpoch = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// withVideoTx runs fn in a transaction that is always rolled back.
func withVideoTx(
	t *testing.T,
	pool *pgxpool.Pool,
	fn func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer),
) {
	t.Helper()

	ctx := t.Context()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin test tx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fn(ctx, tx, repo.NewVideoRepo(db.New(tx), pool, nil))
}

// screenTestTime is the created_at insertScreenVideo gives a video that is
// ageHours old, so a test can build a cursor from it.
func screenTestTime(ageHours int) time.Time {
	return screenTestEpoch.Add(time.Duration(ageHours) * time.Hour)
}

// insertScreenVideo stores a discovered video created ageHours after
// screenTestEpoch. created_at defaults to the transaction start, so without
// this every video of one test would tie.
func insertScreenVideo(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	videos data.VideoRepoer,
	ageHours int,
	caption string,
	hashtags ...string,
) *model.Video {
	t.Helper()

	if hashtags == nil {
		hashtags = []string{}
	}
	externalID := uuid.NewString()
	v := &model.Video{
		VideoURL:         "https://www.tiktok.com/@screen_test/video/" + externalID,
		AuthorUsername:   "screen_test",
		AuthorProfileURL: "https://www.tiktok.com/@screen_test",
		Caption:          caption,
		Hashtags:         hashtags,
		VideoType:        "tiktok",
		ExternalID:       &externalID,
		Origin:           model.VideoOriginDiscovery,
	}
	if _, err := videos.UpsertDiscoveredVideo(ctx, v); err != nil {
		t.Fatalf("failed to insert video: %v", err)
	}

	createdAt := screenTestTime(ageHours)
	if _, err := tx.Exec(
		ctx,
		"update videos set created_at = $2 where id = $1",
		v.ID,
		createdAt,
	); err != nil {
		t.Fatalf("failed to set created_at: %v", err)
	}
	return v
}

// ownedVideos keeps only the videos of got that the test inserted itself.
func ownedVideos(got []*model.Video, owned ...*model.Video) []*model.Video {
	ids := make(map[uuid.UUID]bool, len(owned))
	for _, v := range owned {
		ids[v.ID] = true
	}

	var out []*model.Video
	for _, v := range got {
		if ids[v.ID] {
			out = append(out, v)
		}
	}
	return out
}

func videoIDs(videos []*model.Video) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(videos))
	for _, v := range videos {
		ids = append(ids, v.ID)
	}
	return ids
}

func queryVideoStatus(t *testing.T, ctx context.Context, tx pgx.Tx, id uuid.UUID) *string {
	t.Helper()

	var status *string
	if err := tx.QueryRow(ctx, "select status from videos where id = $1", id).
		Scan(&status); err != nil {
		t.Fatalf("failed to read status: %v", err)
	}
	return status
}

func TestVideoRepo_SetVideoScreenStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("MovesDiscoveredVideo", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertScreenVideo(t, ctx, tx, videos, 0, "claim")

			applied, err := videos.SetVideoScreenStatus(ctx, v.ID, model.VideoStatusScreenedIn)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !applied {
				t.Error("expected the status to be applied")
			}

			status := queryVideoStatus(t, ctx, tx, v.ID)
			if status == nil || *status != string(model.VideoStatusScreenedIn) {
				t.Errorf("status = %v, want screened_in", status)
			}
		})
	})

	t.Run("IsNoOpWhenAlreadyScreened", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertScreenVideo(t, ctx, tx, videos, 0, "claim")
			if _, err := videos.SetVideoScreenStatus(
				ctx,
				v.ID,
				model.VideoStatusScreenedIn,
			); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			applied, err := videos.SetVideoScreenStatus(ctx, v.ID, model.VideoStatusScreenedOut)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if applied {
				t.Error("a screened video must not be changed again")
			}

			status := queryVideoStatus(t, ctx, tx, v.ID)
			if status == nil || *status != string(model.VideoStatusScreenedIn) {
				t.Errorf("status = %v, want it to stay screened_in", status)
			}
		})
	})

	t.Run("UnknownVideo_IsNotApplied", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, _ pgx.Tx, videos data.VideoRepoer) {
			applied, err := videos.SetVideoScreenStatus(
				ctx,
				uuid.New(),
				model.VideoStatusScreenedIn,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if applied {
				t.Error("an unknown video cannot be updated")
			}
		})
	})
}

func TestVideoRepo_GetVideoForScreening(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("ReturnsCaptionAndHashtagsOfDiscoveredVideo", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertScreenVideo(t, ctx, tx, videos, 0, "herbal remedy", "diabetes", "herbal")

			got, err := videos.GetVideoForScreening(ctx, v.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.ID != v.ID {
				t.Errorf("id = %v, want %v", got.ID, v.ID)
			}
			if got.Caption != "herbal remedy" {
				t.Errorf("caption = %q, want %q", got.Caption, "herbal remedy")
			}
			if !slices.Equal(got.Hashtags, []string{"diabetes", "herbal"}) {
				t.Errorf("hashtags = %v, want [diabetes herbal]", got.Hashtags)
			}
		})
	})

	t.Run("ScreenedVideo_IsNotFound", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertScreenVideo(t, ctx, tx, videos, 0, "claim")
			if _, err := videos.SetVideoScreenStatus(
				ctx,
				v.ID,
				model.VideoStatusScreenedIn,
			); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			_, err := videos.GetVideoForScreening(ctx, v.ID)

			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("error = %v, want ErrNotFound", err)
			}
		})
	})

	t.Run("UnknownVideo_IsNotFound", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, _ pgx.Tx, videos data.VideoRepoer) {
			_, err := videos.GetVideoForScreening(ctx, uuid.New())

			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("error = %v, want ErrNotFound", err)
			}
		})
	})
}

// setScreenTestStatus puts a video in status, so a test can build videos of
// every pipeline step.
func setScreenTestStatus(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
	status model.VideoStatus,
) {
	t.Helper()

	if _, err := tx.Exec(
		ctx,
		"update videos set status = $2 where id = $1",
		id,
		string(status),
	); err != nil {
		t.Fatalf("failed to set status: %v", err)
	}
}

func TestVideoRepo_ListVideosByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("ReturnsOnlyTheGivenStatusOldestFirst", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			// Inserted out of order, so the result order can only come from created_at.
			c := insertScreenVideo(t, ctx, tx, videos, 2, "c")
			a := insertScreenVideo(t, ctx, tx, videos, 0, "a")
			b := insertScreenVideo(t, ctx, tx, videos, 1, "b")
			discovered := insertScreenVideo(t, ctx, tx, videos, 3, "discovered")
			transcribed := insertScreenVideo(t, ctx, tx, videos, 4, "transcribed")
			for _, v := range []*model.Video{a, b, c} {
				setScreenTestStatus(t, ctx, tx, v.ID, model.VideoStatusScreenedIn)
			}
			setScreenTestStatus(t, ctx, tx, transcribed.ID, model.VideoStatusTranscribed)

			got, err := videos.ListVideosByStatus(
				ctx, model.VideoStatusScreenedIn, 1000, time.Time{}, uuid.Nil,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			owned := ownedVideos(got, a, b, c, discovered, transcribed)
			if want := []uuid.UUID{a.ID, b.ID, c.ID}; !slices.Equal(videoIDs(owned), want) {
				t.Fatalf("got %v, want only the screened_in videos %v", videoIDs(owned), want)
			}
			if want := screenTestTime(0); !owned[0].CreatedAt.Equal(want) {
				t.Errorf("created_at = %v, want %v", owned[0].CreatedAt, want)
			}

			got, err = videos.ListVideosByStatus(
				ctx, model.VideoStatusTranscribed, 1000, time.Time{}, uuid.Nil,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			owned = ownedVideos(got, a, b, c, discovered, transcribed)
			if want := []uuid.UUID{transcribed.ID}; !slices.Equal(videoIDs(owned), want) {
				t.Errorf("got %v, want only the transcribed video %v", videoIDs(owned), want)
			}
		})
	})

	t.Run("RespectsLimit", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			a := insertScreenVideo(t, ctx, tx, videos, 0, "a")
			b := insertScreenVideo(t, ctx, tx, videos, 1, "b")
			c := insertScreenVideo(t, ctx, tx, videos, 2, "c")
			for _, v := range []*model.Video{a, b, c} {
				setScreenTestStatus(t, ctx, tx, v.ID, model.VideoStatusTranscribed)
			}

			got, err := videos.ListVideosByStatus(
				ctx, model.VideoStatusTranscribed, 2, time.Time{}, uuid.Nil,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want := []uuid.UUID{a.ID, b.ID}; !slices.Equal(videoIDs(got), want) {
				t.Errorf("got %v, want the two oldest videos %v", videoIDs(got), want)
			}
		})
	})

	t.Run("StartsAfterTheCursor", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			a := insertScreenVideo(t, ctx, tx, videos, 0, "a")
			b := insertScreenVideo(t, ctx, tx, videos, 1, "b")
			c := insertScreenVideo(t, ctx, tx, videos, 2, "c")
			for _, v := range []*model.Video{a, b, c} {
				setScreenTestStatus(t, ctx, tx, v.ID, model.VideoStatusScreenedIn)
			}

			got, err := videos.ListVideosByStatus(
				ctx, model.VideoStatusScreenedIn, 1000, screenTestTime(0), a.ID,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			owned := ownedVideos(got, a, b, c)
			if want := []uuid.UUID{b.ID, c.ID}; !slices.Equal(videoIDs(owned), want) {
				t.Errorf("got %v, want %v: the cursor video itself must be skipped",
					videoIDs(owned), want)
			}
		})
	})

	t.Run("CursorWalkVisitsEveryVideoOnceEvenWithTiedCreatedAt", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			// Pairs of videos tie on created_at, so only the id can order them.
			var owned []*model.Video
			for i := range 5 {
				v := insertScreenVideo(t, ctx, tx, videos, i/2, "walk")
				setScreenTestStatus(t, ctx, tx, v.ID, model.VideoStatusScreenedIn)
				owned = append(owned, v)
			}

			var (
				seen    []uuid.UUID
				afterAt time.Time
				afterID uuid.UUID
			)
			for len(seen) < len(owned) {
				page, err := videos.ListVideosByStatus(
					ctx, model.VideoStatusScreenedIn, 2, afterAt, afterID,
				)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(page) == 0 {
					break
				}
				seen = append(seen, videoIDs(ownedVideos(page, owned...))...)
				last := page[len(page)-1]
				afterAt, afterID = last.CreatedAt, last.ID
			}

			want := videoIDs(owned)
			byBytes := func(x, y uuid.UUID) int { return bytes.Compare(x[:], y[:]) }
			slices.SortFunc(seen, byBytes)
			slices.SortFunc(want, byBytes)
			if !slices.Equal(seen, want) {
				t.Errorf("walk saw %v, want every video exactly once: %v", seen, want)
			}
		})
	})

	t.Run("NoVideoInTheStatusReturnsNothingOfTheTestVideos", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			a := insertScreenVideo(t, ctx, tx, videos, 0, "a")

			got, err := videos.ListVideosByStatus(
				ctx, model.VideoStatusTranscribed, 1000, time.Time{}, uuid.Nil,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if owned := ownedVideos(got, a); len(owned) != 0 {
				t.Errorf("got %v, want none: the video is still discovered", videoIDs(owned))
			}
		})
	})

	t.Run("VideoWithoutStatusIsNeverListed", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			waiting := insertScreenVideo(t, ctx, tx, videos, 0, "still waiting")
			// A WhatsApp video has no status and stays out of the pipeline.
			noStatus := insertScreenVideo(t, ctx, tx, videos, 1, "from whatsapp")
			if _, err := tx.Exec(
				ctx,
				"update videos set status = null where id = $1",
				noStatus.ID,
			); err != nil {
				t.Fatalf("failed to clear status: %v", err)
			}

			got, err := videos.ListVideosByStatus(
				ctx, model.VideoStatusDiscovered, 1000, time.Time{}, uuid.Nil,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			owned := ownedVideos(got, waiting, noStatus)
			if want := []uuid.UUID{waiting.ID}; !slices.Equal(videoIDs(owned), want) {
				t.Errorf("got %v, want only %v", videoIDs(owned), want)
			}
		})
	})
}

// noopTranscribeSingleDiscoveredVideoWorker and noopAnalyzeSingleVideoWorker satisfy
// River's client-side check that a job's kind is registered before it can be inserted.
type noopTranscribeSingleDiscoveredVideoWorker struct {
	river.WorkerDefaults[queue.TranscribeSingleDiscoveredVideoJob]
}

func (*noopTranscribeSingleDiscoveredVideoWorker) Work(
	context.Context,
	*river.Job[queue.TranscribeSingleDiscoveredVideoJob],
) error {
	return nil
}

type noopAnalyzeSingleVideoWorker struct {
	river.WorkerDefaults[queue.AnalyzeSingleVideoJob]
}

func (*noopAnalyzeSingleVideoWorker) Work(
	context.Context,
	*river.Job[queue.AnalyzeSingleVideoJob],
) error {
	return nil
}

// pipelineVideoRow is the part of a videos row the pipeline steps change.
type pipelineVideoRow struct {
	status      *string
	failedStage *string
	errorText   *string
	transcript  *string
	mediaURL    *string
	analyzedAt  *time.Time
}

func readPipelineVideo(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
) pipelineVideoRow {
	t.Helper()

	var row pipelineVideoRow
	err := tx.QueryRow(
		ctx,
		`select status, failed_stage, error, transcription_text, media_url, analyzed_at
		 from videos where id = $1`,
		id,
	).Scan(
		&row.status,
		&row.failedStage,
		&row.errorText,
		&row.transcript,
		&row.mediaURL,
		&row.analyzedAt,
	)
	if err != nil {
		t.Fatalf("failed to read video: %v", err)
	}
	return row
}

// insertPipelineVideo stores a video and puts it in status, so a test can
// start at any step of the pipeline.
func insertPipelineVideo(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	videos data.VideoRepoer,
	status model.VideoStatus,
) *model.Video {
	t.Helper()

	v := insertScreenVideo(t, ctx, tx, videos, 0, "claim")
	if _, err := tx.Exec(
		ctx,
		"update videos set status = $2 where id = $1",
		v.ID,
		string(status),
	); err != nil {
		t.Fatalf("failed to set status: %v", err)
	}
	return v
}

func requireStatus(t *testing.T, row pipelineVideoRow, want model.VideoStatus) {
	t.Helper()

	if row.status == nil || *row.status != string(want) {
		t.Errorf("status = %v, want %q", row.status, want)
	}
}

func TestVideoRepo_GetVideoForTranscription(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("ReturnsURLsOfScreenedInVideo", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertPipelineVideo(t, ctx, tx, videos, model.VideoStatusScreenedIn)
			if _, err := tx.Exec(
				ctx,
				"update videos set media_url = $2 where id = $1",
				v.ID,
				"https://media.test/video.mp4",
			); err != nil {
				t.Fatalf("failed to set media url: %v", err)
			}

			got, err := videos.GetVideoForTranscription(ctx, v.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.ID != v.ID {
				t.Errorf("id = %v, want %v", got.ID, v.ID)
			}
			if got.VideoURL != v.VideoURL {
				t.Errorf("video url = %q, want %q", got.VideoURL, v.VideoURL)
			}
			if got.MediaURL == nil || *got.MediaURL != "https://media.test/video.mp4" {
				t.Errorf("media url = %v, want the stored one", got.MediaURL)
			}
		})
	})

	t.Run("MissingMediaURL_IsNil", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertPipelineVideo(t, ctx, tx, videos, model.VideoStatusScreenedIn)

			got, err := videos.GetVideoForTranscription(ctx, v.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.MediaURL != nil {
				t.Errorf("media url = %q, want nil", *got.MediaURL)
			}
		})
	})

	t.Run("VideoInAnotherStatus_IsNotFound", func(t *testing.T) {
		statuses := []model.VideoStatus{
			model.VideoStatusDiscovered,
			model.VideoStatusScreenedOut,
			model.VideoStatusTranscribed,
			model.VideoStatusFailed,
		}
		for _, status := range statuses {
			t.Run(string(status), func(t *testing.T) {
				withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
					v := insertPipelineVideo(t, ctx, tx, videos, status)

					_, err := videos.GetVideoForTranscription(ctx, v.ID)

					if !errors.Is(err, data.ErrNotFound) {
						t.Errorf("error = %v, want ErrNotFound", err)
					}
				})
			})
		}
	})

	t.Run("UnknownVideo_IsNotFound", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, _ pgx.Tx, videos data.VideoRepoer) {
			_, err := videos.GetVideoForTranscription(ctx, uuid.New())

			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("error = %v, want ErrNotFound", err)
			}
		})
	})
}

func TestVideoRepo_SetVideoMediaURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("StoresTheURLOfScreenedInVideo", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertPipelineVideo(t, ctx, tx, videos, model.VideoStatusScreenedIn)

			err := videos.SetVideoMediaURL(ctx, v.ID, "https://media.test/a.mp4")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			row := readPipelineVideo(t, ctx, tx, v.ID)
			if row.mediaURL == nil || *row.mediaURL != "https://media.test/a.mp4" {
				t.Errorf("media url = %v, want the stored one", row.mediaURL)
			}
			requireStatus(t, row, model.VideoStatusScreenedIn)
		})
	})

	t.Run("VideoThatMovedOn_IsLeftAlone", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertPipelineVideo(t, ctx, tx, videos, model.VideoStatusTranscribed)

			err := videos.SetVideoMediaURL(ctx, v.ID, "https://media.test/a.mp4")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			row := readPipelineVideo(t, ctx, tx, v.ID)
			if row.mediaURL != nil {
				t.Errorf("media url = %q, want it untouched", *row.mediaURL)
			}
		})
	})
}

func TestVideoRepo_SetVideoTranscribed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("MovesScreenedInVideoAndStoresTranscript", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertPipelineVideo(t, ctx, tx, videos, model.VideoStatusScreenedIn)

			applied, err := videos.SetVideoTranscribed(ctx, v.ID, "lemon water cures infections")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !applied {
				t.Error("expected the transcript to be applied")
			}

			row := readPipelineVideo(t, ctx, tx, v.ID)
			requireStatus(t, row, model.VideoStatusTranscribed)
			if row.transcript == nil || *row.transcript != "lemon water cures infections" {
				t.Errorf("transcript = %v, want the stored one", row.transcript)
			}
		})
	})

	t.Run("EmptyTranscript_IsStoredAsNull", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertPipelineVideo(t, ctx, tx, videos, model.VideoStatusScreenedIn)

			applied, err := videos.SetVideoTranscribed(ctx, v.ID, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !applied {
				t.Error("a video without speech must still move on")
			}

			row := readPipelineVideo(t, ctx, tx, v.ID)
			requireStatus(t, row, model.VideoStatusTranscribed)
			if row.transcript != nil {
				t.Errorf("transcript = %q, want NULL", *row.transcript)
			}
		})
	})

	t.Run("SecondResultIsDropped", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertPipelineVideo(t, ctx, tx, videos, model.VideoStatusScreenedIn)
			if _, err := videos.SetVideoTranscribed(ctx, v.ID, "first"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			applied, err := videos.SetVideoTranscribed(ctx, v.ID, "second")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if applied {
				t.Error("a transcribed video must not be transcribed again")
			}
			row := readPipelineVideo(t, ctx, tx, v.ID)
			if row.transcript == nil || *row.transcript != "first" {
				t.Errorf("transcript = %v, want the first one to stay", row.transcript)
			}
		})
	})

	t.Run("VideoInAnotherStatus_IsNotApplied", func(t *testing.T) {
		statuses := []model.VideoStatus{
			model.VideoStatusDiscovered,
			model.VideoStatusScreenedOut,
			model.VideoStatusFailed,
		}
		for _, status := range statuses {
			t.Run(string(status), func(t *testing.T) {
				withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
					v := insertPipelineVideo(t, ctx, tx, videos, status)

					applied, err := videos.SetVideoTranscribed(ctx, v.ID, "late result")
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					if applied {
						t.Error("expected the transcript to be dropped")
					}
					row := readPipelineVideo(t, ctx, tx, v.ID)
					requireStatus(t, row, status)
					if row.transcript != nil {
						t.Errorf("transcript = %q, want NULL", *row.transcript)
					}
				})
			})
		}
	})

	t.Run("UnknownVideo_IsNotApplied", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, _ pgx.Tx, videos data.VideoRepoer) {
			applied, err := videos.SetVideoTranscribed(ctx, uuid.New(), "text")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if applied {
				t.Error("an unknown video must not be applied")
			}
		})
	})
}

func TestVideoRepo_GetVideoForAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("ReturnsTranscriptAndPublishTimeOfTranscribedVideo", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertPipelineVideo(t, ctx, tx, videos, model.VideoStatusTranscribed)
			publishedAt := time.Date(2026, time.November, 1, 10, 0, 0, 0, time.UTC)
			if _, err := tx.Exec(
				ctx,
				"update videos set transcription_text = $2, video_created_at = $3 where id = $1",
				v.ID,
				"lemon water cures infections",
				publishedAt,
			); err != nil {
				t.Fatalf("failed to set transcript: %v", err)
			}

			got, err := videos.GetVideoForAnalysis(ctx, v.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.ID != v.ID {
				t.Errorf("id = %v, want %v", got.ID, v.ID)
			}
			if got.TranscriptionText == nil ||
				*got.TranscriptionText != "lemon water cures infections" {
				t.Errorf("transcript = %v, want the stored one", got.TranscriptionText)
			}
			if !got.VideoCreatedAt.Equal(publishedAt) {
				t.Errorf("video created at = %v, want %v", got.VideoCreatedAt, publishedAt)
			}
		})
	})

	t.Run("VideoWithoutSpeech_HasNilTranscript", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			v := insertPipelineVideo(t, ctx, tx, videos, model.VideoStatusTranscribed)

			got, err := videos.GetVideoForAnalysis(ctx, v.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.TranscriptionText != nil {
				t.Errorf("transcript = %q, want nil", *got.TranscriptionText)
			}
		})
	})

	t.Run("VideoInAnotherStatus_IsNotFound", func(t *testing.T) {
		statuses := []model.VideoStatus{
			model.VideoStatusScreenedIn,
			model.VideoStatusVerified,
			model.VideoStatusClustered,
			model.VideoStatusFailed,
		}
		for _, status := range statuses {
			t.Run(string(status), func(t *testing.T) {
				withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
					v := insertPipelineVideo(t, ctx, tx, videos, status)

					_, err := videos.GetVideoForAnalysis(ctx, v.ID)

					if !errors.Is(err, data.ErrNotFound) {
						t.Errorf("error = %v, want ErrNotFound", err)
					}
				})
			})
		}
	})

	t.Run("UnknownVideo_IsNotFound", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, _ pgx.Tx, videos data.VideoRepoer) {
			_, err := videos.GetVideoForAnalysis(ctx, uuid.New())

			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("error = %v, want ErrNotFound", err)
			}
		})
	})
}

func TestVideoRepo_SetVideoAnalyzed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("MovesTranscribedVideoAndStampsAnalyzedAt", func(t *testing.T) {
		finalStatuses := []model.VideoStatus{
			model.VideoStatusVerified,
			model.VideoStatusOutOfScope,
		}
		for _, status := range finalStatuses {
			t.Run(string(status), func(t *testing.T) {
				withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
					v := insertPipelineVideo(t, ctx, tx, videos, model.VideoStatusTranscribed)

					applied, err := videos.SetVideoAnalyzed(ctx, v.ID, status)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if !applied {
						t.Error("expected the status to be applied")
					}

					row := readPipelineVideo(t, ctx, tx, v.ID)
					requireStatus(t, row, status)
					if row.analyzedAt == nil {
						t.Error("analyzed_at must be set")
					}
				})
			})
		}
	})

	t.Run("VideoInAnotherStatus_IsNotApplied", func(t *testing.T) {
		statuses := []model.VideoStatus{
			model.VideoStatusScreenedIn,
			model.VideoStatusVerified,
			model.VideoStatusClustered,
			model.VideoStatusFailed,
		}
		for _, status := range statuses {
			t.Run(string(status), func(t *testing.T) {
				withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
					v := insertPipelineVideo(t, ctx, tx, videos, status)

					applied, err := videos.SetVideoAnalyzed(ctx, v.ID, model.VideoStatusOutOfScope)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					if applied {
						t.Error("expected the status to be dropped")
					}
					row := readPipelineVideo(t, ctx, tx, v.ID)
					requireStatus(t, row, status)
					if row.analyzedAt != nil {
						t.Errorf("analyzed_at = %v, want NULL", row.analyzedAt)
					}
				})
			})
		}
	})

	t.Run("UnknownVideo_IsNotApplied", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, _ pgx.Tx, videos data.VideoRepoer) {
			applied, err := videos.SetVideoAnalyzed(ctx, uuid.New(), model.VideoStatusVerified)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if applied {
				t.Error("an unknown video must not be applied")
			}
		})
	})
}

func TestVideoRepo_SetVideoFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("MarksVideoThatIsStillInFromStatus", func(t *testing.T) {
		tests := []struct {
			name  string
			from  model.VideoStatus
			stage string
		}{
			{"screen", model.VideoStatusDiscovered, model.VideoFailedStageScreen},
			{"transcribe", model.VideoStatusScreenedIn, model.VideoFailedStageTranscribe},
			{"analyze", model.VideoStatusTranscribed, model.VideoFailedStageAnalyze},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
					v := insertPipelineVideo(t, ctx, tx, videos, tt.from)

					applied, err := videos.SetVideoFailed(ctx, v.ID, tt.from, tt.stage, "asr down")
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if !applied {
						t.Error("expected the failure to be applied")
					}

					row := readPipelineVideo(t, ctx, tx, v.ID)
					requireStatus(t, row, model.VideoStatusFailed)
					if row.failedStage == nil || *row.failedStage != tt.stage {
						t.Errorf("failed_stage = %v, want %q", row.failedStage, tt.stage)
					}
					if row.errorText == nil || *row.errorText != "asr down" {
						t.Errorf("error = %v, want %q", row.errorText, "asr down")
					}
				})
			})
		}
	})

	t.Run("VideoThatMovedOn_IsNotApplied", func(t *testing.T) {
		tests := []struct {
			name   string
			status model.VideoStatus
			from   model.VideoStatus
		}{
			{"already screened", model.VideoStatusScreenedIn, model.VideoStatusDiscovered},
			{"already transcribed", model.VideoStatusTranscribed, model.VideoStatusScreenedIn},
			{"already verified", model.VideoStatusVerified, model.VideoStatusTranscribed},
			{"already failed", model.VideoStatusFailed, model.VideoStatusScreenedIn},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
					v := insertPipelineVideo(t, ctx, tx, videos, tt.status)

					applied, err := videos.SetVideoFailed(
						ctx, v.ID, tt.from, model.VideoFailedStageTranscribe, "late failure",
					)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					if applied {
						t.Error("a video that moved on must not be marked as failed")
					}
					row := readPipelineVideo(t, ctx, tx, v.ID)
					requireStatus(t, row, tt.status)
					if row.failedStage != nil || row.errorText != nil {
						t.Errorf("failure = %v/%v, want none", row.failedStage, row.errorText)
					}
				})
			})
		}
	})

	t.Run("UnknownVideo_IsNotApplied", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, _ pgx.Tx, videos data.VideoRepoer) {
			applied, err := videos.SetVideoFailed(
				ctx, uuid.New(), model.VideoStatusScreenedIn, model.VideoFailedStageTranscribe, "x",
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if applied {
				t.Error("an unknown video must not be applied")
			}
		})
	})
}

// TestVideoRepo_EnqueueVideoPipelineJobs commits real river_job rows (the
// enqueue runs on its own connection when no transaction is open), so every
// subtest deletes the rows it created.
func TestVideoRepo_EnqueueVideoPipelineJobs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, q := testhelpers.SetupTestDB(t)
	defer pool.Close()

	// River validates at Insert-time that the job kind is registered in the
	// client's Workers bundle, even though this test never starts the client.
	workers := river.NewWorkers()
	river.AddWorker(workers, &noopTranscribeSingleDiscoveredVideoWorker{})
	river.AddWorker(workers, &noopAnalyzeSingleVideoWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("failed to construct river client: %v", err)
	}
	videos := repo.NewVideoRepo(q, pool, func() *river.Client[pgx.Tx] { return riverClient })

	tests := []struct {
		name      string
		kind      queue.BackgroundJobType
		queueName string
		enqueue   func(ctx context.Context, r data.VideoRepoer, id uuid.UUID) (bool, error)
	}{
		{
			"TranscribeVideo",
			queue.JobTypeTranscribeSingleDiscoveredVideo,
			queue.QueueTranscription,
			func(ctx context.Context, r data.VideoRepoer, id uuid.UUID) (bool, error) {
				return r.EnqueueTranscribeVideoJob(ctx, id)
			},
		},
		{
			"AnalyzeVideo",
			queue.JobTypeAnalyzeSingleVideo,
			queue.QueueAnalysis,
			func(ctx context.Context, r data.VideoRepoer, id uuid.UUID) (bool, error) {
				return r.EnqueueAnalyzeVideoJob(ctx, id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			// newVideoID returns an id whose jobs are deleted when the subtest ends.
			newVideoID := func(t *testing.T) uuid.UUID {
				t.Helper()

				id := uuid.New()
				t.Cleanup(func() {
					if _, err := pool.Exec(
						context.Background(),
						"delete from river_job where kind = $1 and args->>'video_id' = $2",
						string(tt.kind), id.String(),
					); err != nil {
						t.Errorf("failed to delete river jobs: %v", err)
					}
				})
				return id
			}

			countJobs := func(t *testing.T, id uuid.UUID) int {
				t.Helper()

				var jobs int
				err := pool.QueryRow(
					ctx,
					`select count(*) from river_job
					 where kind = $1 and queue = $2 and args->>'video_id' = $3`,
					string(tt.kind), tt.queueName, id.String(),
				).Scan(&jobs)
				if err != nil {
					t.Fatalf("failed to count river jobs: %v", err)
				}
				return jobs
			}

			t.Run("InsertsOneJobIntoItsQueue", func(t *testing.T) {
				id := newVideoID(t)

				enqueued, err := tt.enqueue(ctx, videos, id)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if !enqueued {
					t.Error("expected a new job")
				}
				if jobs := countJobs(t, id); jobs != 1 {
					t.Errorf("jobs = %d, want 1", jobs)
				}
			})

			t.Run("DuplicateWhileWaiting_IsSkipped", func(t *testing.T) {
				id := newVideoID(t)
				if _, err := tt.enqueue(ctx, videos, id); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				enqueued, err := tt.enqueue(ctx, videos, id)
				if err != nil {
					t.Fatalf("a duplicate must not be an error: %v", err)
				}

				if enqueued {
					t.Error("expected the duplicate to be reported as not enqueued")
				}
				if jobs := countJobs(t, id); jobs != 1 {
					t.Errorf("jobs = %d, want 1", jobs)
				}
			})

			t.Run("CompletedJob_DoesNotBlockANewOne", func(t *testing.T) {
				id := newVideoID(t)
				if _, err := tt.enqueue(ctx, videos, id); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if _, err := pool.Exec(
					ctx,
					`update river_job set state = 'completed', finalized_at = now()
					 where kind = $1 and args->>'video_id' = $2`,
					string(tt.kind), id.String(),
				); err != nil {
					t.Fatalf("failed to complete the job: %v", err)
				}

				enqueued, err := tt.enqueue(ctx, videos, id)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if !enqueued {
					t.Error("a finished job must not block a new one")
				}
				if jobs := countJobs(t, id); jobs != 2 {
					t.Errorf("jobs = %d, want 2", jobs)
				}
			})

			t.Run("InsideWithTx_CommitsWithTheTransaction", func(t *testing.T) {
				id := newVideoID(t)

				err := videos.WithTx(ctx, func(r data.VideoRepoer) error {
					_, err := tt.enqueue(ctx, r, id)
					return err
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if jobs := countJobs(t, id); jobs != 1 {
					t.Errorf("jobs = %d, want 1", jobs)
				}
			})

			t.Run("InsideWithTx_RollsBackWithTheTransaction", func(t *testing.T) {
				id := newVideoID(t)
				boom := errors.New("boom")

				err := videos.WithTx(ctx, func(r data.VideoRepoer) error {
					enqueued, err := tt.enqueue(ctx, r, id)
					if err != nil {
						return err
					}
					if !enqueued {
						t.Error("expected a new job inside the transaction")
					}
					return boom
				})

				if !errors.Is(err, boom) {
					t.Fatalf("error = %v, want the transaction error", err)
				}
				if jobs := countJobs(t, id); jobs != 0 {
					t.Errorf(
						"jobs = %d, want 0: the job must roll back with the status change",
						jobs,
					)
				}
			})
		})
	}
}
