package handler_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/labstack/echo/v4"
)

const adminInfluencersPath = "/api/v1/admin/influencers"

var influencerCreatedAt = time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)

func newAdminInfluencerApp(t *testing.T, repo data.InfluencerRepoer) *echo.Echo {
	t.Helper()
	return newAdminAppWith(t, core.WithInfluencerRepo(repo))
}

func influencerFixture(id uuid.UUID) *model.Influencer {
	return &model.Influencer{
		ID:         id,
		Platform:   "tiktok",
		Username:   "someone",
		ProfileURL: "https://www.tiktok.com/@someone",
		Source:     model.InfluencerSourceSeed,
		IsActive:   true,
		CreatedAt:  influencerCreatedAt,
		UpdatedAt:  influencerCreatedAt,
	}
}

func influencerRepoCallCount(repo *mockrepo.InfluencerRepoerMock) int {
	return len(repo.PaginateCalls()) + len(repo.GetByIDCalls()) +
		len(repo.CreateCalls()) + len(repo.SetActiveCalls())
}

func TestAdminInfluencerRoutes_Authorization(t *testing.T) {
	id := uuid.NewString()
	body := marshalAdminBody(t, map[string]any{"username": "someone", "is_active": true})

	routes := []struct{ name, method, path string }{
		{"List", http.MethodGet, adminInfluencersPath},
		{"Create", http.MethodPost, adminInfluencersPath},
		{"Get", http.MethodGet, adminInfluencersPath + "/" + id},
		{"SetActive", http.MethodPatch, adminInfluencersPath + "/" + id},
	}
	callers := []struct {
		name          string
		authorization string
		want          int
	}{
		{"WithoutToken", "", http.StatusUnauthorized},
		{"InvalidToken", "Bearer not-a-token", http.StatusUnauthorized},
		{"AsUser", bearerFor(t, model.RoleUser), http.StatusForbidden},
	}

	for _, rt := range routes {
		for _, c := range callers {
			t.Run(rt.name+"_"+c.name, func(t *testing.T) {
				repo := &mockrepo.InfluencerRepoerMock{}
				app := newAdminInfluencerApp(t, repo)

				rec := serveAdminReq(app, rt.method, rt.path, c.authorization, body)
				expectAdminStatus(t, rec, c.want)

				if n := influencerRepoCallCount(repo); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	}
}

func TestListInfluencersHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	newRepo := func(
		gotQuery *data.SupportsPagination,
		gotFilter *data.InfluencerListFilter,
	) *mockrepo.InfluencerRepoerMock {
		return &mockrepo.InfluencerRepoerMock{
			PaginateFunc: func(
				_ context.Context,
				q data.SupportsPagination,
				f data.InfluencerListFilter,
			) (*dto.PaginationResults[model.Influencer], error) {
				*gotQuery, *gotFilter = q, f
				return &dto.PaginationResults[model.Influencer]{
					Page: 2, PerPage: 5, TotalItems: 6,
					Items: []model.Influencer{*influencerFixture(uuid.New())},
				}, nil
			},
		}
	}

	t.Run("Success_PassesPagingAndFilters", func(t *testing.T) {
		var gotQuery data.SupportsPagination
		var gotFilter data.InfluencerListFilter
		app := newAdminInfluencerApp(t, newRepo(&gotQuery, &gotFilter))

		rec := serveAdminReq(app, http.MethodGet,
			adminInfluencersPath+"?page=2&per_page=5&is_active=false&source=discovered&search=some",
			admin, nil)
		expectAdminStatus(t, rec, http.StatusOK)

		got := decodeAdminResp[dto.PaginationResults[model.Influencer]](t, rec)
		if got.Page != 2 || got.PerPage != 5 || got.TotalItems != 6 || len(got.Items) != 1 {
			t.Errorf("unexpected page in response: %+v", got)
		}
		if gotQuery.Limit() != 5 || gotQuery.Offset() != 5 {
			t.Errorf(
				"expected limit 5 / offset 5, got %d / %d",
				gotQuery.Limit(),
				gotQuery.Offset(),
			)
		}
		if gotFilter.IsActive == nil || *gotFilter.IsActive ||
			gotFilter.Source != "discovered" || gotFilter.Search != "some" {
			t.Errorf("unexpected filter: %+v", gotFilter)
		}
	})

	t.Run("Success_NoQuery_UsesDefaults", func(t *testing.T) {
		var gotQuery data.SupportsPagination
		var gotFilter data.InfluencerListFilter
		app := newAdminInfluencerApp(t, newRepo(&gotQuery, &gotFilter))

		rec := serveAdminReq(app, http.MethodGet, adminInfluencersPath, admin, nil)
		expectAdminStatus(t, rec, http.StatusOK)

		if gotQuery.Limit() != 20 || gotQuery.Offset() != 0 {
			t.Errorf("expected default limit 20 / offset 0, got %d / %d",
				gotQuery.Limit(), gotQuery.Offset())
		}
		if gotFilter.IsActive != nil || gotFilter.Source != "" || gotFilter.Search != "" {
			t.Errorf("expected no filter, got %+v", gotFilter)
		}
	})

	t.Run("InvalidQuery", func(t *testing.T) {
		tests := []struct {
			name  string
			query string
			want  int
		}{
			{"PerPage_TooLarge", "?per_page=101", http.StatusUnprocessableEntity},
			{"Source_Unknown", "?source=other", http.StatusUnprocessableEntity},
			{
				"Search_TooLong",
				"?search=" + strings.Repeat("a", 101),
				http.StatusUnprocessableEntity,
			},
			{"IsActive_NotABool", "?is_active=maybe", http.StatusBadRequest},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &mockrepo.InfluencerRepoerMock{}
				app := newAdminInfluencerApp(t, repo)

				rec := serveAdminReq(app, http.MethodGet, adminInfluencersPath+tt.query, admin, nil)
				expectAdminStatus(t, rec, tt.want)

				if n := len(repo.PaginateCalls()); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	})

	for name, tt := range adminRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.InfluencerRepoerMock{
				PaginateFunc: func(
					context.Context, data.SupportsPagination, data.InfluencerListFilter,
				) (*dto.PaginationResults[model.Influencer], error) {
					return nil, tt.err
				},
			}
			app := newAdminInfluencerApp(t, repo)

			rec := serveAdminReq(app, http.MethodGet, adminInfluencersPath, admin, nil)
			expectAdminStatus(t, rec, tt.want)
		})
	}
}

func TestGetInfluencerHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	t.Run("Success", func(t *testing.T) {
		id := uuid.New()
		var gotID uuid.UUID
		repo := &mockrepo.InfluencerRepoerMock{
			GetByIDFunc: func(_ context.Context, id uuid.UUID) (*model.Influencer, error) {
				gotID = id
				return influencerFixture(id), nil
			},
		}
		app := newAdminInfluencerApp(t, repo)

		rec := serveAdminReq(app, http.MethodGet, adminInfluencersPath+"/"+id.String(), admin, nil)
		expectAdminStatus(t, rec, http.StatusOK)

		got := decodeAdminResp[model.Influencer](t, rec)
		if got.ID != id || got.Username != "someone" || !got.IsActive {
			t.Errorf("unexpected influencer in response: %+v", got)
		}
		if gotID != id {
			t.Errorf("expected the repo to be asked for %s, got %s", id, gotID)
		}
	})

	t.Run("InvalidID", func(t *testing.T) {
		repo := &mockrepo.InfluencerRepoerMock{}
		app := newAdminInfluencerApp(t, repo)

		rec := serveAdminReq(app, http.MethodGet, adminInfluencersPath+"/not-a-uuid", admin, nil)
		expectAdminStatus(t, rec, http.StatusBadRequest)

		if n := len(repo.GetByIDCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	for name, tt := range adminRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.InfluencerRepoerMock{
				GetByIDFunc: func(context.Context, uuid.UUID) (*model.Influencer, error) {
					return nil, tt.err
				},
			}
			app := newAdminInfluencerApp(t, repo)

			rec := serveAdminReq(app, http.MethodGet,
				adminInfluencersPath+"/"+uuid.NewString(), admin, nil)
			expectAdminStatus(t, rec, tt.want)
		})
	}
}

func TestCreateInfluencerHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	newRepo := func(saved **model.Influencer) *mockrepo.InfluencerRepoerMock {
		return &mockrepo.InfluencerRepoerMock{
			CreateFunc: func(_ context.Context, i *model.Influencer) error {
				*saved = i
				i.ID = uuid.New()
				i.IsActive = true
				i.CreatedAt, i.UpdatedAt = influencerCreatedAt, influencerCreatedAt
				return nil
			},
		}
	}

	valid := map[string]string{
		"WithAt":       "@some.one_1",
		"WithoutAt":    "some.one_1",
		"MinLength":    "ab",
		"MaxLength":    strings.Repeat("a", 24),
		"SurroundedBy": "  @some.one_1  ",
	}
	for name, username := range valid {
		t.Run("Success_"+name, func(t *testing.T) {
			var saved *model.Influencer
			app := newAdminInfluencerApp(t, newRepo(&saved))

			rec := serveAdminReq(app, http.MethodPost, adminInfluencersPath, admin,
				marshalAdminBody(t, map[string]any{"username": username}))
			expectAdminStatus(t, rec, http.StatusCreated)

			want := strings.TrimPrefix(strings.TrimSpace(username), "@")
			got := decodeAdminResp[model.Influencer](t, rec)
			if got.ID == uuid.Nil || got.Username != want ||
				got.ProfileURL != "https://www.tiktok.com/@"+want ||
				got.Platform != "tiktok" || got.Source != model.InfluencerSourceSeed {
				t.Errorf("unexpected influencer in response: %+v", got)
			}
			if saved == nil || saved.Username != want {
				t.Errorf("unexpected influencer sent to repo: %+v", saved)
			}
		})
	}

	t.Run("ValidationFailed", func(t *testing.T) {
		bodies := map[string]map[string]any{
			"Username_Missing":       {},
			"Username_Empty":         {"username": ""},
			"Username_OnlyAt":        {"username": "@"},
			"Username_TooShort":      {"username": "a"},
			"Username_TooLong":       {"username": strings.Repeat("a", 25)},
			"Username_HasSpace":      {"username": "some one"},
			"Username_HasDash":       {"username": "some-one"},
			"Username_DoubleAt":      {"username": "@@someone"},
			"Username_AtInTheMiddle": {"username": "some@one"},
		}
		for name, body := range bodies {
			t.Run(name, func(t *testing.T) {
				repo := &mockrepo.InfluencerRepoerMock{}
				app := newAdminInfluencerApp(t, repo)

				rec := serveAdminReq(app, http.MethodPost, adminInfluencersPath, admin,
					marshalAdminBody(t, body))
				expectAdminStatus(t, rec, http.StatusUnprocessableEntity)

				if n := len(repo.CreateCalls()); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		app := newAdminInfluencerApp(t, &mockrepo.InfluencerRepoerMock{})

		rec := serveAdminReq(app, http.MethodPost, adminInfluencersPath, admin, []byte(`{`))
		expectAdminStatus(t, rec, http.StatusBadRequest)
	})

	for name, tt := range adminRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.InfluencerRepoerMock{
				CreateFunc: func(context.Context, *model.Influencer) error { return tt.err },
			}
			app := newAdminInfluencerApp(t, repo)

			rec := serveAdminReq(app, http.MethodPost, adminInfluencersPath, admin,
				marshalAdminBody(t, map[string]any{"username": "someone"}))
			expectAdminStatus(t, rec, tt.want)
		})
	}
}

func TestSetInfluencerActiveHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	for _, active := range []bool{true, false} {
		t.Run("Success", func(t *testing.T) {
			id := uuid.New()
			var gotID uuid.UUID
			var gotActive bool
			repo := &mockrepo.InfluencerRepoerMock{
				SetActiveFunc: func(
					_ context.Context, id uuid.UUID, active bool,
				) (*model.Influencer, error) {
					gotID, gotActive = id, active
					i := influencerFixture(id)
					i.IsActive = active
					return i, nil
				},
			}
			app := newAdminInfluencerApp(t, repo)

			rec := serveAdminReq(app, http.MethodPatch, adminInfluencersPath+"/"+id.String(),
				admin, marshalAdminBody(t, map[string]any{"is_active": active}))
			expectAdminStatus(t, rec, http.StatusOK)

			got := decodeAdminResp[model.Influencer](t, rec)
			if got.ID != id || got.IsActive != active {
				t.Errorf("unexpected influencer in response: %+v", got)
			}
			if gotID != id || gotActive != active {
				t.Errorf("unexpected repo call: id=%s active=%v", gotID, gotActive)
			}
		})
	}

	t.Run("InvalidID", func(t *testing.T) {
		repo := &mockrepo.InfluencerRepoerMock{}
		app := newAdminInfluencerApp(t, repo)

		rec := serveAdminReq(app, http.MethodPatch, adminInfluencersPath+"/not-a-uuid", admin,
			marshalAdminBody(t, map[string]any{"is_active": true}))
		expectAdminStatus(t, rec, http.StatusBadRequest)

		if n := len(repo.SetActiveCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	t.Run("ValidationFailed_IsActiveMissing", func(t *testing.T) {
		repo := &mockrepo.InfluencerRepoerMock{}
		app := newAdminInfluencerApp(t, repo)

		rec := serveAdminReq(app, http.MethodPatch, adminInfluencersPath+"/"+uuid.NewString(),
			admin, marshalAdminBody(t, map[string]any{}))
		expectAdminStatus(t, rec, http.StatusUnprocessableEntity)

		if n := len(repo.SetActiveCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		app := newAdminInfluencerApp(t, &mockrepo.InfluencerRepoerMock{})

		rec := serveAdminReq(app, http.MethodPatch, adminInfluencersPath+"/"+uuid.NewString(),
			admin, []byte(`{`))
		expectAdminStatus(t, rec, http.StatusBadRequest)
	})

	for name, tt := range adminRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.InfluencerRepoerMock{
				SetActiveFunc: func(context.Context, uuid.UUID, bool) (*model.Influencer, error) {
					return nil, tt.err
				},
			}
			app := newAdminInfluencerApp(t, repo)

			rec := serveAdminReq(app, http.MethodPatch, adminInfluencersPath+"/"+uuid.NewString(),
				admin, marshalAdminBody(t, map[string]any{"is_active": true}))
			expectAdminStatus(t, rec, tt.want)
		})
	}
}

const dashboardInfluencersPath = "/api/v1/influencers"

func TestDashboardInfluencersHandler(t *testing.T) {
	user := bearerFor(t, model.RoleUser)

	t.Run("WithoutToken_Unauthorized", func(t *testing.T) {
		repo := &mockrepo.InfluencerRepoerMock{}
		app := newAdminInfluencerApp(t, repo)

		rec := serveAdminReq(app, http.MethodGet, dashboardInfluencersPath, "", nil)
		expectAdminStatus(t, rec, http.StatusUnauthorized)

		if n := influencerRepoCallCount(repo); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	t.Run("AsUser_ListsSortedByPosts", func(t *testing.T) {
		var gotFilter data.InfluencerListFilter
		repo := &mockrepo.InfluencerRepoerMock{
			PaginateFunc: func(
				_ context.Context,
				_ data.SupportsPagination,
				f data.InfluencerListFilter,
			) (*dto.PaginationResults[model.Influencer], error) {
				gotFilter = f
				return &dto.PaginationResults[model.Influencer]{
					Page: 1, PerPage: 10, TotalItems: 1,
					Items: []model.Influencer{*influencerFixture(uuid.New())},
				}, nil
			},
		}
		app := newAdminInfluencerApp(t, repo)

		rec := serveAdminReq(app, http.MethodGet,
			dashboardInfluencersPath+"?sort=posts&is_active=true&per_page=10", user, nil)
		expectAdminStatus(t, rec, http.StatusOK)

		got := decodeAdminResp[dto.PaginationResults[model.Influencer]](t, rec)
		if got.TotalItems != 1 || len(got.Items) != 1 {
			t.Errorf("unexpected page in response: %+v", got)
		}
		if gotFilter.Sort != "posts" || gotFilter.IsActive == nil || !*gotFilter.IsActive {
			t.Errorf("unexpected filter: %+v", gotFilter)
		}
	})

	t.Run("InvalidSort_Unprocessable", func(t *testing.T) {
		repo := &mockrepo.InfluencerRepoerMock{}
		app := newAdminInfluencerApp(t, repo)

		rec := serveAdminReq(
			app,
			http.MethodGet,
			dashboardInfluencersPath+"?sort=oldest",
			user,
			nil,
		)
		expectAdminStatus(t, rec, http.StatusUnprocessableEntity)

		if n := len(repo.PaginateCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})
}
