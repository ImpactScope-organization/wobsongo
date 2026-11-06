package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/labstack/echo/v4"
)

const adminTopicsPath = "/api/v1/admin/topics"

var topicCreatedAt = time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)

// adminRepoErrors are the errors a repo can return and the status each one
// must become. Shared by the topic and influencer tests.
var adminRepoErrors = map[string]struct {
	err  error
	want int
}{
	"NotFound": {fmt.Errorf("get: %w", data.ErrNotFound), http.StatusNotFound},
	"Conflict": {fmt.Errorf("save: %w", data.ErrConflict), http.StatusConflict},
	"Internal": {fmt.Errorf("query: %w", data.ErrInternal), http.StatusInternalServerError},
	"Unmapped": {errors.New("connection reset"), http.StatusInternalServerError},
}

// newAdminAppWith builds the app with one repo option and without signing
// requests in, so each test picks its own caller.
func newAdminAppWith(t *testing.T, opt core.AppOption) *echo.Echo {
	t.Helper()
	t.Setenv("AGENT_ENABLED", "false")
	useTestJWTSecret(t)

	app := core.NewApp(testhelpers.NewEcho(), internal.NewConfig(), opt)
	return app.Echo()
}

func serveAdminReq(
	app *echo.Echo,
	method, path, authorization string,
	body []byte,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if authorization != "" {
		req.Header.Set(echo.HeaderAuthorization, authorization)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func expectAdminStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("expected status %d, got %d: %s", want, rec.Code, rec.Body.String())
	}
}

func decodeAdminResp[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var resp testhelpers.APIResponse[T]
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	return resp.Data
}

func newAdminTopicApp(t *testing.T, repo data.TopicRepoer) *echo.Echo {
	t.Helper()
	return newAdminAppWith(t, core.WithTopicRepo(repo))
}

func topicFixture(id uuid.UUID) *model.Topic {
	return &model.Topic{
		ID:          id,
		Name:        "Health",
		Description: "Claims about diseases",
		IsActive:    true,
		CreatedAt:   topicCreatedAt,
		UpdatedAt:   topicCreatedAt,
	}
}

// withTopicTx makes WithTx run fn on the mock itself, like a transaction that
// commits, and lets the lock succeed unless a test overrides it.
func withTopicTx(repo *mockrepo.TopicRepoerMock) *mockrepo.TopicRepoerMock {
	repo.WithTxFunc = func(_ context.Context, fn func(data.TopicRepoer) error) error {
		return fn(repo)
	}
	if repo.LockActiveGuardFunc == nil {
		repo.LockActiveGuardFunc = func(context.Context) error { return nil }
	}
	return repo
}

func topicRepoCallCount(repo *mockrepo.TopicRepoerMock) int {
	return len(repo.PaginateCalls()) + len(repo.GetByIDCalls()) +
		len(repo.CreateCalls()) + len(repo.UpdateCalls()) +
		len(repo.WithTxCalls()) + len(repo.LockActiveGuardCalls())
}

func TestAdminTopicRoutes_Authorization(t *testing.T) {
	id := uuid.NewString()
	body := marshalAdminBody(t, map[string]any{"name": "Health", "is_active": true})

	routes := []struct{ name, method, path string }{
		{"List", http.MethodGet, adminTopicsPath},
		{"Create", http.MethodPost, adminTopicsPath},
		{"Get", http.MethodGet, adminTopicsPath + "/" + id},
		{"Update", http.MethodPut, adminTopicsPath + "/" + id},
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
				repo := &mockrepo.TopicRepoerMock{}
				app := newAdminTopicApp(t, repo)

				rec := serveAdminReq(app, rt.method, rt.path, c.authorization, body)
				expectAdminStatus(t, rec, c.want)

				if n := topicRepoCallCount(repo); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	}
}

func TestListTopicsHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	newRepo := func(
		gotQuery *data.SupportsPagination,
		gotFilter *data.AdminListFilter,
	) *mockrepo.TopicRepoerMock {
		return &mockrepo.TopicRepoerMock{
			PaginateFunc: func(
				_ context.Context,
				q data.SupportsPagination,
				f data.AdminListFilter,
			) (*dto.PaginationResults[model.Topic], error) {
				*gotQuery, *gotFilter = q, f
				return &dto.PaginationResults[model.Topic]{
					Page: 2, PerPage: 5, TotalItems: 6,
					Items: []model.Topic{*topicFixture(uuid.New())},
				}, nil
			},
		}
	}

	t.Run("Success_PassesPagingAndFilters", func(t *testing.T) {
		var gotQuery data.SupportsPagination
		var gotFilter data.AdminListFilter
		app := newAdminTopicApp(t, newRepo(&gotQuery, &gotFilter))

		rec := serveAdminReq(app, http.MethodGet,
			adminTopicsPath+"?page=2&per_page=5&is_active=true&search=heal", admin, nil)
		expectAdminStatus(t, rec, http.StatusOK)

		got := decodeAdminResp[dto.PaginationResults[model.Topic]](t, rec)
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
		if gotFilter.IsActive == nil || !*gotFilter.IsActive || gotFilter.Search != "heal" {
			t.Errorf("unexpected filter: %+v", gotFilter)
		}
	})

	t.Run("Success_InactiveFilter", func(t *testing.T) {
		var gotQuery data.SupportsPagination
		var gotFilter data.AdminListFilter
		app := newAdminTopicApp(t, newRepo(&gotQuery, &gotFilter))

		rec := serveAdminReq(app, http.MethodGet, adminTopicsPath+"?is_active=false", admin, nil)
		expectAdminStatus(t, rec, http.StatusOK)

		if gotFilter.IsActive == nil || *gotFilter.IsActive {
			t.Errorf("expected is_active=false to reach the repo, got %+v", gotFilter.IsActive)
		}
	})

	t.Run("Success_NoQuery_UsesDefaults", func(t *testing.T) {
		var gotQuery data.SupportsPagination
		var gotFilter data.AdminListFilter
		app := newAdminTopicApp(t, newRepo(&gotQuery, &gotFilter))

		rec := serveAdminReq(app, http.MethodGet, adminTopicsPath, admin, nil)
		expectAdminStatus(t, rec, http.StatusOK)

		if gotQuery.Limit() != 20 || gotQuery.Offset() != 0 {
			t.Errorf("expected default limit 20 / offset 0, got %d / %d",
				gotQuery.Limit(), gotQuery.Offset())
		}
		if gotFilter.IsActive != nil || gotFilter.Search != "" {
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
			{"Page_Negative", "?page=-1", http.StatusUnprocessableEntity},
			{
				"Search_TooLong",
				"?search=" + strings.Repeat("a", 101),
				http.StatusUnprocessableEntity,
			},
			{"PerPage_NotANumber", "?per_page=abc", http.StatusBadRequest},
			{"IsActive_NotABool", "?is_active=maybe", http.StatusBadRequest},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &mockrepo.TopicRepoerMock{}
				app := newAdminTopicApp(t, repo)

				rec := serveAdminReq(app, http.MethodGet, adminTopicsPath+tt.query, admin, nil)
				expectAdminStatus(t, rec, tt.want)

				if n := len(repo.PaginateCalls()); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	})

	for name, tt := range adminRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.TopicRepoerMock{
				PaginateFunc: func(
					context.Context, data.SupportsPagination, data.AdminListFilter,
				) (*dto.PaginationResults[model.Topic], error) {
					return nil, tt.err
				},
			}
			app := newAdminTopicApp(t, repo)

			rec := serveAdminReq(app, http.MethodGet, adminTopicsPath, admin, nil)
			expectAdminStatus(t, rec, tt.want)
		})
	}
}

func TestGetTopicHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	t.Run("Success", func(t *testing.T) {
		id := uuid.New()
		var gotID uuid.UUID
		repo := &mockrepo.TopicRepoerMock{
			GetByIDFunc: func(_ context.Context, id uuid.UUID) (*model.Topic, error) {
				gotID = id
				return topicFixture(id), nil
			},
		}
		app := newAdminTopicApp(t, repo)

		rec := serveAdminReq(app, http.MethodGet, adminTopicsPath+"/"+id.String(), admin, nil)
		expectAdminStatus(t, rec, http.StatusOK)

		got := decodeAdminResp[model.Topic](t, rec)
		if got.ID != id || got.Name != "Health" || !got.IsActive {
			t.Errorf("unexpected topic in response: %+v", got)
		}
		if gotID != id {
			t.Errorf("expected the repo to be asked for %s, got %s", id, gotID)
		}
	})

	t.Run("InvalidID", func(t *testing.T) {
		repo := &mockrepo.TopicRepoerMock{}
		app := newAdminTopicApp(t, repo)

		rec := serveAdminReq(app, http.MethodGet, adminTopicsPath+"/not-a-uuid", admin, nil)
		expectAdminStatus(t, rec, http.StatusBadRequest)

		if n := len(repo.GetByIDCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	for name, tt := range adminRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.TopicRepoerMock{
				GetByIDFunc: func(context.Context, uuid.UUID) (*model.Topic, error) {
					return nil, tt.err
				},
			}
			app := newAdminTopicApp(t, repo)

			rec := serveAdminReq(app, http.MethodGet,
				adminTopicsPath+"/"+uuid.NewString(), admin, nil)
			expectAdminStatus(t, rec, tt.want)
		})
	}
}

func TestCreateTopicHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	newRepo := func(saved **model.Topic) *mockrepo.TopicRepoerMock {
		return &mockrepo.TopicRepoerMock{
			CreateFunc: func(_ context.Context, topic *model.Topic) error {
				*saved = topic
				topic.ID = uuid.New()
				topic.CreatedAt, topic.UpdatedAt = topicCreatedAt, topicCreatedAt
				return nil
			},
		}
	}

	t.Run("Success_TrimsAndDefaultsToActive", func(t *testing.T) {
		var saved *model.Topic
		app := newAdminTopicApp(t, newRepo(&saved))

		rec := serveAdminReq(app, http.MethodPost, adminTopicsPath, admin,
			marshalAdminBody(t, map[string]any{"name": "  Health ", "description": " Diseases "}))
		expectAdminStatus(t, rec, http.StatusCreated)

		got := decodeAdminResp[model.Topic](t, rec)
		if got.ID == uuid.Nil || got.Name != "Health" || got.Description != "Diseases" ||
			!got.IsActive {
			t.Errorf("unexpected topic in response: %+v", got)
		}
	})

	t.Run("Success_ExplicitInactive", func(t *testing.T) {
		var saved *model.Topic
		app := newAdminTopicApp(t, newRepo(&saved))

		rec := serveAdminReq(app, http.MethodPost, adminTopicsPath, admin,
			marshalAdminBody(t, map[string]any{"name": "Health", "is_active": false}))
		expectAdminStatus(t, rec, http.StatusCreated)

		if saved == nil || saved.IsActive {
			t.Errorf("expected an inactive topic to reach the repo, got %+v", saved)
		}
	})

	t.Run("ValidationFailed", func(t *testing.T) {
		bodies := map[string]map[string]any{
			"Name_Missing":        {"description": "x"},
			"Name_Blank":          {"name": "   "},
			"Name_TooShort":       {"name": "a"},
			"Name_TooLong":        {"name": strings.Repeat("a", 101)},
			"Description_TooLong": {"name": "Health", "description": strings.Repeat("a", 1001)},
		}
		for name, body := range bodies {
			t.Run(name, func(t *testing.T) {
				repo := &mockrepo.TopicRepoerMock{}
				app := newAdminTopicApp(t, repo)

				rec := serveAdminReq(app, http.MethodPost, adminTopicsPath, admin,
					marshalAdminBody(t, body))
				expectAdminStatus(t, rec, http.StatusUnprocessableEntity)

				if n := len(repo.CreateCalls()); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		app := newAdminTopicApp(t, &mockrepo.TopicRepoerMock{})

		rec := serveAdminReq(app, http.MethodPost, adminTopicsPath, admin, []byte(`{`))
		expectAdminStatus(t, rec, http.StatusBadRequest)
	})

	for name, tt := range adminRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.TopicRepoerMock{
				CreateFunc: func(context.Context, *model.Topic) error { return tt.err },
			}
			app := newAdminTopicApp(t, repo)

			rec := serveAdminReq(app, http.MethodPost, adminTopicsPath, admin,
				marshalAdminBody(t, map[string]any{"name": "Health"}))
			expectAdminStatus(t, rec, tt.want)
		})
	}
}

func TestUpdateTopicHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)
	validBody := map[string]any{"name": " Health ", "description": "Desc", "is_active": false}

	t.Run("Success_ReplacesFields", func(t *testing.T) {
		id := uuid.New()
		var saved *model.Topic
		repo := withTopicTx(&mockrepo.TopicRepoerMock{
			UpdateFunc: func(_ context.Context, topic *model.Topic) error {
				saved = topic
				return nil
			},
		})
		app := newAdminTopicApp(t, repo)

		rec := serveAdminReq(app, http.MethodPut, adminTopicsPath+"/"+id.String(), admin,
			marshalAdminBody(t, validBody))
		expectAdminStatus(t, rec, http.StatusOK)

		got := decodeAdminResp[model.Topic](t, rec)
		if got.ID != id || got.Name != "Health" || got.IsActive {
			t.Errorf("unexpected topic in response: %+v", got)
		}
		if saved == nil || saved.ID != id {
			t.Errorf("expected the path ID to reach the repo, got %+v", saved)
		}
	})

	t.Run("LastActiveTopic_Conflict", func(t *testing.T) {
		repo := withTopicTx(&mockrepo.TopicRepoerMock{
			UpdateFunc: func(context.Context, *model.Topic) error { return data.ErrNotFound },
			GetByIDFunc: func(_ context.Context, id uuid.UUID) (*model.Topic, error) {
				return topicFixture(id), nil
			},
		})
		app := newAdminTopicApp(t, repo)

		rec := serveAdminReq(app, http.MethodPut, adminTopicsPath+"/"+uuid.NewString(), admin,
			marshalAdminBody(t, validBody))
		expectAdminStatus(t, rec, http.StatusConflict)

		if !strings.Contains(rec.Body.String(), "at least one topic must stay active") {
			t.Errorf("expected the last-active message, got %s", rec.Body.String())
		}
	})

	t.Run("UnknownTopic_NotFound", func(t *testing.T) {
		repo := withTopicTx(&mockrepo.TopicRepoerMock{
			UpdateFunc: func(context.Context, *model.Topic) error { return data.ErrNotFound },
			GetByIDFunc: func(context.Context, uuid.UUID) (*model.Topic, error) {
				return nil, data.ErrNotFound
			},
		})
		app := newAdminTopicApp(t, repo)

		rec := serveAdminReq(app, http.MethodPut, adminTopicsPath+"/"+uuid.NewString(), admin,
			marshalAdminBody(t, validBody))
		expectAdminStatus(t, rec, http.StatusNotFound)
	})

	t.Run("InvalidID", func(t *testing.T) {
		repo := &mockrepo.TopicRepoerMock{}
		app := newAdminTopicApp(t, repo)

		rec := serveAdminReq(app, http.MethodPut, adminTopicsPath+"/not-a-uuid", admin,
			marshalAdminBody(t, validBody))
		expectAdminStatus(t, rec, http.StatusBadRequest)

		if n := len(repo.UpdateCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	t.Run("ValidationFailed", func(t *testing.T) {
		bodies := map[string]map[string]any{
			"IsActive_Missing": {"name": "Health"},
			"Name_Missing":     {"is_active": true},
			"Name_Blank":       {"name": "   ", "is_active": true},
			"Name_TooShort":    {"name": "a", "is_active": true},
		}
		for name, body := range bodies {
			t.Run(name, func(t *testing.T) {
				repo := &mockrepo.TopicRepoerMock{}
				app := newAdminTopicApp(t, repo)

				rec := serveAdminReq(app, http.MethodPut, adminTopicsPath+"/"+uuid.NewString(),
					admin, marshalAdminBody(t, body))
				expectAdminStatus(t, rec, http.StatusUnprocessableEntity)

				if n := len(repo.UpdateCalls()); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		app := newAdminTopicApp(t, &mockrepo.TopicRepoerMock{})

		rec := serveAdminReq(app, http.MethodPut, adminTopicsPath+"/"+uuid.NewString(),
			admin, []byte(`{`))
		expectAdminStatus(t, rec, http.StatusBadRequest)
	})

	for _, name := range []string{"Conflict", "Internal", "Unmapped"} {
		tt := adminRepoErrors[name]
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := withTopicTx(&mockrepo.TopicRepoerMock{
				UpdateFunc: func(context.Context, *model.Topic) error { return tt.err },
			})
			app := newAdminTopicApp(t, repo)

			rec := serveAdminReq(app, http.MethodPut, adminTopicsPath+"/"+uuid.NewString(),
				admin, marshalAdminBody(t, validBody))
			expectAdminStatus(t, rec, tt.want)
		})
	}

	t.Run("LockError_Internal", func(t *testing.T) {
		repo := withTopicTx(&mockrepo.TopicRepoerMock{
			LockActiveGuardFunc: func(context.Context) error {
				return fmt.Errorf("lock: %w", data.ErrInternal)
			},
		})
		app := newAdminTopicApp(t, repo)

		rec := serveAdminReq(app, http.MethodPut, adminTopicsPath+"/"+uuid.NewString(), admin,
			marshalAdminBody(t, validBody))
		expectAdminStatus(t, rec, http.StatusInternalServerError)

		if n := len(repo.UpdateCalls()); n != 0 {
			t.Errorf("expected no update after a failed lock, got %d", n)
		}
	})
}
