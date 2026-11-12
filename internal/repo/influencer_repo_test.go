package repo_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/jackc/pgx/v5"
)

func insertAdminInfluencer(
	ctx context.Context,
	t *testing.T,
	tx pgx.Tx,
	username, source string,
	active bool,
	emptyChecks int,
) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := tx.QueryRow(ctx,
		`insert into influencers
		   (platform, username, profile_url, source, is_active, empty_check_count, next_check_at)
		 values ('tiktok', $1, $2, $3, $4, $5, NOW() + interval '30 days')
		 returning id`,
		username, "https://www.tiktok.com/@"+username, source, active, emptyChecks,
	).Scan(&id)
	if err != nil {
		t.Fatalf("failed to insert influencer %q: %v", username, err)
	}
	return id
}

func influencerUsernames(items []model.Influencer) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Username)
	}
	return names
}

// insertCreatorPosts adds count tiktok videos of an influencer. publishedAt
// is the video's own publish time; nil leaves video_created_at empty, so the
// row's created_at is used instead.
func insertCreatorPosts(
	ctx context.Context,
	t *testing.T,
	tx pgx.Tx,
	influencerID uuid.UUID,
	username string,
	count int,
	publishedAt *time.Time,
) {
	t.Helper()

	for range count {
		_, err := tx.Exec(ctx,
			`insert into videos
			   (video_url, author_username, video_type, influencer_id, origin, status,
			    video_created_at)
			 values ($1, $2, 'tiktok', $3, 'monitor', 'discovered', $4)`,
			"https://www.tiktok.com/@"+username+"/video/"+uuid.NewString(),
			username, influencerID, publishedAt,
		)
		if err != nil {
			t.Fatalf("failed to insert video of %q: %v", username, err)
		}
	}
}

func topCreatorUsernames(items []model.TopCreator) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Username)
	}
	return strings.Join(names, ",")
}

func TestInfluencerAdminRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	newRepo := func(q *db.Queries) data.InfluencerRepoer {
		return repo.NewInfluencerRepo(q, pool)
	}

	t.Run("Create_FillsStoredFields", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			influencers := newRepo(q)
			username := "zz_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
			influencer := &model.Influencer{
				Platform:   "tiktok",
				Username:   username,
				ProfileURL: "https://www.tiktok.com/@" + username,
				Source:     model.InfluencerSourceSeed,
			}

			if err := influencers.Create(ctx, influencer); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if influencer.ID == uuid.Nil || influencer.CreatedAt.IsZero() ||
				influencer.UpdatedAt.IsZero() || !influencer.IsActive ||
				influencer.EmptyCheckCount != 0 {
				t.Errorf("unexpected stored fields: %+v", influencer)
			}

			got, err := influencers.GetByID(ctx, influencer.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Username != username || got.Platform != "tiktok" ||
				got.Source != model.InfluencerSourceSeed ||
				got.ProfileURL != "https://www.tiktok.com/@"+username {
				t.Errorf("unexpected influencer: %+v", got)
			}
		})
	})

	t.Run("Create_SameUsernameIgnoringCase_Conflicts", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			username := "zzDup" + strings.ReplaceAll(uuid.NewString()[:6], "-", "")
			insertAdminInfluencer(ctx, t, tx, username, "seed", true, 0)

			err := newRepo(q).Create(ctx, &model.Influencer{
				Platform:   "tiktok",
				Username:   strings.ToLower(username),
				ProfileURL: "https://www.tiktok.com/@" + strings.ToLower(username),
				Source:     model.InfluencerSourceSeed,
			})
			if !errors.Is(err, data.ErrConflict) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})
	})

	t.Run("SetActive_Deactivate_KeepsCounterAndSchedule", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			id := insertAdminInfluencer(ctx, t, tx,
				"zz"+strings.ReplaceAll(uuid.NewString()[:8], "-", ""), "seed", true, 2)

			got, err := newRepo(q).SetActive(ctx, id, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.IsActive || got.EmptyCheckCount != 2 {
				t.Errorf("expected inactive with counter 2, got %+v", got)
			}
			if got.NextCheckAt == nil || !got.NextCheckAt.After(time.Now().Add(24*time.Hour)) {
				t.Errorf("expected the schedule to stay in the future, got %v", got.NextCheckAt)
			}
		})
	})

	t.Run("SetActive_Reactivate_ResetsCounterAndSchedulesNow", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			id := insertAdminInfluencer(ctx, t, tx,
				"zz"+strings.ReplaceAll(uuid.NewString()[:8], "-", ""), "discovered", false, 5)

			got, err := newRepo(q).SetActive(ctx, id, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.IsActive || got.EmptyCheckCount != 0 {
				t.Errorf("expected active with counter 0, got %+v", got)
			}
			if got.NextCheckAt == nil || got.NextCheckAt.After(time.Now().Add(time.Minute)) {
				t.Errorf("expected a check to be due right away, got %v", got.NextCheckAt)
			}
		})
	})

	t.Run("SetActive_AlreadyActive_ChangesNothing", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			id := insertAdminInfluencer(ctx, t, tx,
				"zz"+strings.ReplaceAll(uuid.NewString()[:8], "-", ""), "seed", true, 2)

			got, err := newRepo(q).SetActive(ctx, id, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.IsActive || got.EmptyCheckCount != 2 {
				t.Errorf("expected the counter to stay 2, got %+v", got)
			}
			if got.NextCheckAt == nil || !got.NextCheckAt.After(time.Now().Add(24*time.Hour)) {
				t.Errorf("expected the schedule to stay in the future, got %v", got.NextCheckAt)
			}
		})
	})

	t.Run("SetActive_UnknownID_NotFound", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			_, err := newRepo(q).SetActive(ctx, uuid.New(), true)
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	})

	t.Run("Paginate_FiltersAndOrdersNewestFirst", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			token := "zz" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
			insertAdminInfluencer(ctx, t, tx, token+"_a", "seed", true, 0)
			insertAdminInfluencer(ctx, t, tx, token+"_b", "discovered", true, 0)
			insertAdminInfluencer(ctx, t, tx, token+"_c", "discovered", false, 0)
			influencers := newRepo(q)
			paging := &dto.PaginationDTO{Page: 1, PerPage: 10}

			all, err := influencers.Paginate(ctx, paging,
				data.InfluencerListFilter{AdminListFilter: data.AdminListFilter{Search: token}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if all.TotalItems != 3 || len(all.Items) != 3 {
				t.Fatalf("expected 3 influencers, got %v", influencerUsernames(all.Items))
			}
			// Rows are inserted in one transaction, so created_at is equal and
			// the id (uuidv7, increasing) breaks the tie: newest first.
			if all.Items[0].Username != token+"_c" || all.Items[2].Username != token+"_a" {
				t.Errorf("expected newest first, got %v", influencerUsernames(all.Items))
			}

			upper, err := influencers.Paginate(ctx, paging, data.InfluencerListFilter{
				AdminListFilter: data.AdminListFilter{Search: strings.ToUpper(token)},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if upper.TotalItems != 3 {
				t.Errorf("expected a case-insensitive search to find 3, got %d", upper.TotalItems)
			}

			discovered, err := influencers.Paginate(ctx, paging, data.InfluencerListFilter{
				AdminListFilter: data.AdminListFilter{Search: token},
				Source:          "discovered",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if discovered.TotalItems != 2 {
				t.Errorf("expected 2 discovered, got %v", influencerUsernames(discovered.Items))
			}

			inactive := false
			discoveredInactive, err := influencers.Paginate(ctx, paging, data.InfluencerListFilter{
				AdminListFilter: data.AdminListFilter{Search: token, IsActive: &inactive},
				Source:          "discovered",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if discoveredInactive.TotalItems != 1 ||
				discoveredInactive.Items[0].Username != token+"_c" {
				t.Errorf("expected only %s_c, got %v",
					token, influencerUsernames(discoveredInactive.Items))
			}

			page2, err := influencers.Paginate(ctx, &dto.PaginationDTO{Page: 2, PerPage: 2},
				data.InfluencerListFilter{AdminListFilter: data.AdminListFilter{Search: token}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if page2.Page != 2 || page2.PerPage != 2 || page2.TotalItems != 3 ||
				len(page2.Items) != 1 {
				t.Errorf("unexpected second page: %+v", page2)
			}
		})
	})

	t.Run("Paginate_SortByPostsRanksPostsThenFollowers", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			token := "zz" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
			// Inserted in this order, so the newest is _many_high, then _few.
			counts := []struct {
				suffix    string
				posts     int64
				followers int64
			}{
				{"_many_low", 5, 100},
				{"_few", 2, 10_000},
				{"_many_high", 5, 900},
			}
			for _, c := range counts {
				id := insertAdminInfluencer(ctx, t, tx, token+c.suffix, "seed", true, 0)
				_, err := tx.Exec(
					ctx,
					`update influencers set tracked_video_count = $1, followers_count = $2 where id = $3`,
					c.posts,
					c.followers,
					id,
				)
				if err != nil {
					t.Fatalf("failed to set counts of %s: %v", c.suffix, err)
				}
			}
			influencers := newRepo(q)
			paging := &dto.PaginationDTO{Page: 1, PerPage: 10}
			filter := data.InfluencerListFilter{
				AdminListFilter: data.AdminListFilter{Search: token},
			}

			newest, err := influencers.Paginate(ctx, paging, filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			wantNewest := []string{token + "_many_high", token + "_few", token + "_many_low"}
			if got := influencerUsernames(
				newest.Items,
			); strings.Join(
				got,
				",",
			) != strings.Join(
				wantNewest,
				",",
			) {
				t.Errorf("expected newest first without a sort, got %v", got)
			}

			filter.Sort = "posts"
			byPosts, err := influencers.Paginate(ctx, paging, filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			wantByPosts := []string{token + "_many_high", token + "_many_low", token + "_few"}
			if got := influencerUsernames(
				byPosts.Items,
			); strings.Join(
				got,
				",",
			) != strings.Join(
				wantByPosts,
				",",
			) {
				t.Errorf("expected most posts first, then most followers, got %v", got)
			}
		})
	})

	t.Run("Paginate_SearchMatchesNickname", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			token := "zz" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
			id := insertAdminInfluencer(ctx, t, tx,
				"zz"+strings.ReplaceAll(uuid.NewString()[:8], "-", ""), "seed", true, 0)
			if _, err := tx.Exec(
				ctx,
				`update influencers set nickname = $1 where id = $2`,
				"Nick "+token,
				id,
			); err != nil {
				t.Fatalf("failed to set nickname: %v", err)
			}

			got, err := newRepo(q).Paginate(ctx, &dto.PaginationDTO{Page: 1, PerPage: 10},
				data.InfluencerListFilter{AdminListFilter: data.AdminListFilter{Search: token}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.TotalItems != 1 || got.Items[0].ID != id {
				t.Errorf("expected the nickname match, got %+v", got)
			}
		})
	})

	t.Run("Paginate_NoMatch_ReturnsEmptyItems", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			got, err := newRepo(q).Paginate(ctx, &dto.PaginationDTO{Page: 1, PerPage: 10},
				data.InfluencerListFilter{
					AdminListFilter: data.AdminListFilter{Search: "zz-" + uuid.NewString()},
				})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Items == nil || len(got.Items) != 0 || got.TotalItems != 0 {
				t.Errorf("expected an empty, non-nil page, got %+v", got)
			}
		})
	})

	t.Run("TopCreators_CountsPostsInWindowAndRanksThem", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			token := "zz" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
			now := time.Now()
			recent, old := now.Add(-24*time.Hour), now.Add(-10*24*time.Hour)

			setups := []struct {
				suffix      string
				recentPosts int
				oldPosts    int
				followers   int64
			}{
				// Same posts in the window as _many_high, but more followers
				// would not matter: it has fewer, so _many_high ranks first.
				// Its old posts must not count.
				{"_many_low", 3, 4, 100},
				{"_few", 1, 0, 10_000},
				{"_many_high", 3, 0, 900},
				{"_only_old", 0, 2, 50},
				{"_no_posts", 0, 0, 50},
			}
			for _, s := range setups {
				username := token + s.suffix
				id := insertAdminInfluencer(ctx, t, tx, username, "seed", true, 0)
				if _, err := tx.Exec(ctx,
					`update influencers
					 set followers_count = $1, nickname = $2, avatar_url = $3
					 where id = $4`,
					s.followers, "Nick "+s.suffix, "https://img.example/"+s.suffix, id,
				); err != nil {
					t.Fatalf("failed to set profile of %s: %v", s.suffix, err)
				}
				insertCreatorPosts(ctx, t, tx, id, username, s.recentPosts, &recent)
				insertCreatorPosts(ctx, t, tx, id, username, s.oldPosts, &old)
			}

			got, err := newRepo(q).PaginateTopCreators(ctx,
				&dto.PaginationDTO{Page: 1, PerPage: 10},
				data.TopCreatorFilter{
					AdminListFilter: data.AdminListFilter{Search: token},
					Since:           now.Add(-7 * 24 * time.Hour),
				})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Ties on posts are broken by followers. Creators without a post
			// in the window are left out, and so are they from the total.
			want := strings.Join(
				[]string{token + "_many_high", token + "_many_low", token + "_few"}, ",")
			if names := topCreatorUsernames(got.Items); names != want {
				t.Fatalf("expected %s, got %s", want, names)
			}
			if got.TotalItems != 3 {
				t.Errorf("expected a total of 3, got %d", got.TotalItems)
			}

			first := got.Items[0]
			if first.PostsCount != 3 || got.Items[1].PostsCount != 3 ||
				got.Items[2].PostsCount != 1 {
				t.Errorf("expected only posts in the window to count, got %+v", got.Items)
			}
			if first.Platform != "tiktok" || first.FollowersCount != 900 ||
				first.Nickname != "Nick _many_high" ||
				first.AvatarURL != "https://img.example/_many_high" ||
				first.ProfileURL != "https://www.tiktok.com/@"+token+"_many_high" {
				t.Errorf("unexpected profile fields: %+v", first)
			}
		})
	})

	t.Run("TopCreators_NoVideoCreatedAt_FallsBackToCreatedAt", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			token := "zz" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
			old := time.Now().Add(-10 * 24 * time.Hour)

			// Published time unknown: the time the row was stored (now) is used.
			unknown := token + "_unknown"
			unknownID := insertAdminInfluencer(ctx, t, tx, unknown, "seed", true, 0)
			insertCreatorPosts(ctx, t, tx, unknownID, unknown, 2, nil)

			// Published long ago, only stored now: the publish time wins.
			stale := token + "_stale"
			staleID := insertAdminInfluencer(ctx, t, tx, stale, "seed", true, 0)
			insertCreatorPosts(ctx, t, tx, staleID, stale, 1, &old)

			got, err := newRepo(q).PaginateTopCreators(ctx,
				&dto.PaginationDTO{Page: 1, PerPage: 10},
				data.TopCreatorFilter{
					AdminListFilter: data.AdminListFilter{Search: token},
					Since:           time.Now().Add(-24 * time.Hour),
				})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if names := topCreatorUsernames(got.Items); names != unknown {
				t.Fatalf("expected only %s, got %s", unknown, names)
			}
			if got.Items[0].PostsCount != 2 {
				t.Errorf("expected 2 posts, got %d", got.Items[0].PostsCount)
			}
		})
	})

	t.Run("TopCreators_FiltersByActiveAndSearch", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			token := "zz" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
			nickToken := "nk" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
			recent := time.Now().Add(-time.Hour)

			active, inactive := token+"_active", token+"_inactive"
			activeID := insertAdminInfluencer(ctx, t, tx, active, "seed", true, 0)
			inactiveID := insertAdminInfluencer(ctx, t, tx, inactive, "seed", false, 0)
			insertCreatorPosts(ctx, t, tx, activeID, active, 2, &recent)
			insertCreatorPosts(ctx, t, tx, inactiveID, inactive, 1, &recent)
			if _, err := tx.Exec(ctx,
				`update influencers set nickname = $1 where id = $2`,
				"Nick "+nickToken, inactiveID,
			); err != nil {
				t.Fatalf("failed to set nickname: %v", err)
			}

			isActive, isInactive := true, false
			cases := []struct {
				name   string
				filter data.AdminListFilter
				want   string
			}{
				{"NoFilter", data.AdminListFilter{Search: token}, active + "," + inactive},
				{"OnlyActive", data.AdminListFilter{Search: token, IsActive: &isActive}, active},
				{
					"OnlyInactive",
					data.AdminListFilter{Search: token, IsActive: &isInactive},
					inactive,
				},
				{
					"SearchMatchesNickname",
					data.AdminListFilter{Search: strings.ToUpper(nickToken)},
					inactive,
				},
			}
			for _, c := range cases {
				got, err := newRepo(q).PaginateTopCreators(ctx,
					&dto.PaginationDTO{Page: 1, PerPage: 10},
					data.TopCreatorFilter{
						AdminListFilter: c.filter,
						Since:           time.Now().Add(-24 * time.Hour),
					})
				if err != nil {
					t.Fatalf("%s: unexpected error: %v", c.name, err)
				}
				if names := topCreatorUsernames(got.Items); names != c.want {
					t.Errorf("%s: expected %s, got %s", c.name, c.want, names)
				}
				if got.TotalItems != len(got.Items) {
					t.Errorf(
						"%s: expected total %d, got %d",
						c.name,
						len(got.Items),
						got.TotalItems,
					)
				}
			}
		})
	})

	t.Run("TopCreators_Paginates_TotalCoversAllPages", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			token := "zz" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
			recent := time.Now().Add(-time.Hour)

			for i, posts := range []int{3, 2, 1} {
				username := token + "_" + strconv.Itoa(i)
				id := insertAdminInfluencer(ctx, t, tx, username, "seed", true, 0)
				insertCreatorPosts(ctx, t, tx, id, username, posts, &recent)
			}

			got, err := newRepo(q).PaginateTopCreators(ctx,
				&dto.PaginationDTO{Page: 2, PerPage: 2},
				data.TopCreatorFilter{
					AdminListFilter: data.AdminListFilter{Search: token},
					Since:           time.Now().Add(-24 * time.Hour),
				})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Page != 2 || got.PerPage != 2 || got.TotalItems != 3 ||
				topCreatorUsernames(got.Items) != token+"_2" {
				t.Errorf("unexpected second page: %+v", got)
			}
		})
	})

	t.Run("TopCreators_NoMatch_ReturnsEmptyItems", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			got, err := newRepo(q).PaginateTopCreators(ctx,
				&dto.PaginationDTO{Page: 1, PerPage: 10},
				data.TopCreatorFilter{
					AdminListFilter: data.AdminListFilter{Search: "zz-" + uuid.NewString()},
					Since:           time.Now().Add(-24 * time.Hour),
				})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Items == nil || len(got.Items) != 0 || got.TotalItems != 0 {
				t.Errorf("expected an empty, non-nil page, got %+v", got)
			}
		})
	})

	t.Run("Upsert_SameUsernameIgnoringCase_UpdatesExistingRow", func(t *testing.T) {
		withRollbackTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			username := "zzUp" + strings.ReplaceAll(uuid.NewString()[:6], "-", "")
			id := insertAdminInfluencer(ctx, t, tx, username, "seed", true, 0)

			m := &model.Influencer{
				Platform:   "tiktok",
				Username:   strings.ToLower(username),
				ProfileURL: "https://www.tiktok.com/@" + strings.ToLower(username),
				Nickname:   "Updated",
				Source:     model.InfluencerSourceDiscovered,
			}
			if err := newRepo(q).UpsertInfluencer(ctx, m); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.ID != id {
				t.Errorf("expected the existing row %v to be updated, got %v", id, m.ID)
			}

			var count int
			err := tx.QueryRow(ctx,
				`select count(*) from influencers
			 where platform = 'tiktok' and lower(username) = lower($1)`, username,
			).Scan(&count)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != 1 {
				t.Errorf("expected one row, got %d", count)
			}
		})
	})
}
