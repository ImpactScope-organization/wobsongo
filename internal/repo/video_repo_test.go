package repo_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

	createdAt := screenTestEpoch.Add(time.Duration(ageHours) * time.Hour)
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

func TestVideoRepo_ListDiscoveredVideos(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("ReturnsOldestFirstWithCaptionAndHashtags", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			// Inserted out of order, so the result order can only come from created_at.
			c := insertScreenVideo(t, ctx, tx, videos, 2, "recipe")
			a := insertScreenVideo(t, ctx, tx, videos, 0, "herbal remedy", "diabetes", "herbal")
			b := insertScreenVideo(t, ctx, tx, videos, 1, "no hashtags")

			got, err := videos.ListDiscoveredVideos(ctx, 1000, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			owned := ownedVideos(got, a, b, c)
			if want := []uuid.UUID{a.ID, b.ID, c.ID}; !slices.Equal(videoIDs(owned), want) {
				t.Fatalf("order = %v, want %v", videoIDs(owned), want)
			}
			if owned[0].Caption != "herbal remedy" {
				t.Errorf("caption = %q, want %q", owned[0].Caption, "herbal remedy")
			}
			if !slices.Equal(owned[0].Hashtags, []string{"diabetes", "herbal"}) {
				t.Errorf("hashtags = %v, want [diabetes herbal]", owned[0].Hashtags)
			}
		})
	})

	t.Run("RespectsLimit", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			a := insertScreenVideo(t, ctx, tx, videos, 0, "a")
			b := insertScreenVideo(t, ctx, tx, videos, 1, "b")
			insertScreenVideo(t, ctx, tx, videos, 2, "c")

			got, err := videos.ListDiscoveredVideos(ctx, 2, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want := []uuid.UUID{a.ID, b.ID}; !slices.Equal(videoIDs(got), want) {
				t.Errorf("got %v, want the two oldest videos %v", videoIDs(got), want)
			}
		})
	})

	t.Run("SkipsExcludedIDs", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			a := insertScreenVideo(t, ctx, tx, videos, 0, "a")
			b := insertScreenVideo(t, ctx, tx, videos, 1, "b")
			c := insertScreenVideo(t, ctx, tx, videos, 2, "c")

			got, err := videos.ListDiscoveredVideos(ctx, 2, []uuid.UUID{a.ID})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want := []uuid.UUID{b.ID, c.ID}; !slices.Equal(videoIDs(got), want) {
				t.Errorf("got %v, want %v", videoIDs(got), want)
			}
		})
	})

	t.Run("EmptyExcludeListMatchesEveryVideo", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			a := insertScreenVideo(t, ctx, tx, videos, 0, "a")

			got, err := videos.ListDiscoveredVideos(ctx, 1000, []uuid.UUID{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(ownedVideos(got, a)) != 1 {
				t.Error("an empty exclude list must not hide any video")
			}
		})
	})

	t.Run("OnlyReturnsDiscoveredVideos", func(t *testing.T) {
		withVideoTx(t, pool, func(ctx context.Context, tx pgx.Tx, videos data.VideoRepoer) {
			screened := insertScreenVideo(t, ctx, tx, videos, 0, "already screened")
			waiting := insertScreenVideo(t, ctx, tx, videos, 1, "still waiting")
			noStatus := insertScreenVideo(t, ctx, tx, videos, 2, "from whatsapp")

			if _, err := videos.SetVideoScreenStatus(
				ctx,
				screened.ID,
				model.VideoStatusScreenedIn,
			); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// A WhatsApp video has no status and stays out of the pipeline.
			if _, err := tx.Exec(
				ctx,
				"update videos set status = null where id = $1",
				noStatus.ID,
			); err != nil {
				t.Fatalf("failed to clear status: %v", err)
			}

			got, err := videos.ListDiscoveredVideos(ctx, 1000, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			owned := ownedVideos(got, screened, waiting, noStatus)
			if want := []uuid.UUID{waiting.ID}; !slices.Equal(videoIDs(owned), want) {
				t.Errorf("got %v, want only %v", videoIDs(owned), want)
			}
		})
	})
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
