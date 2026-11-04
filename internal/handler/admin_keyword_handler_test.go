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

const adminKeywordsPath = "/api/v1/admin/keywords"

var keywordCreatedAt = time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)

// keywordRepoErrors are the errors a repo can return and the status each one
// must become.
var keywordRepoErrors = map[string]struct {
	err  error
	want int
}{
	"NotFound": {fmt.Errorf("get keyword: %w", data.ErrNotFound), http.StatusNotFound},
	"Conflict": {fmt.Errorf("save keyword: %w", data.ErrConflict), http.StatusConflict},
	"Internal": {fmt.Errorf("query keyword: %w", data.ErrInternal), http.StatusInternalServerError},
	"Unmapped": {errors.New("connection reset"), http.StatusInternalServerError},
}

// newAdminKeywordTestApp builds the app without signing requests in, so each
// test picks its own caller (admin, plain user, or nobody).
func newAdminKeywordTestApp(t *testing.T, repo data.KeywordRepoer) *echo.Echo {
	t.Helper()
	// The agent is enabled by default and then needs an LLM endpoint to be
	// built. It is not under test here.
	t.Setenv("AGENT_ENABLED", "false")
	useTestJWTSecret(t)

	app := core.NewApp(
		testhelpers.NewEcho(),
		internal.NewConfig(),
		core.WithKeywordRepo(repo),
	)
	return app.Echo()
}

func serveAdminKeyword(
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

func expectKeywordStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("expected status %d, got %d: %s", want, rec.Code, rec.Body.String())
	}
}

func decodeKeywordData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var resp testhelpers.APIResponse[T]
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	return resp.Data
}

func keywordFixture(id uuid.UUID) *model.DiscoveryKeyword {
	return &model.DiscoveryKeyword{
		ID:           id,
		Keyword:      "vaccin",
		Language:     model.LanguageFrench,
		LanguageCode: "fr",
		IsActive:     true,
		CreatedAt:    keywordCreatedAt,
		UpdatedAt:    keywordCreatedAt,
	}
}

func keywordRepoCallCount(repo *mockrepo.KeywordRepoerMock) int {
	return len(repo.PaginateCalls()) + len(repo.GetByIDCalls()) +
		len(repo.CreateCalls()) + len(repo.UpdateCalls()) + len(repo.DeleteCalls())
}

func TestAdminKeywordRoutes_Authorization(t *testing.T) {
	id := uuid.NewString()
	body := marshalAdminBody(t, map[string]any{
		"keyword": "vaccin", "language": "fr", "is_active": true,
	})

	routes := []struct{ name, method, path string }{
		{"List", http.MethodGet, adminKeywordsPath},
		{"Create", http.MethodPost, adminKeywordsPath},
		{"Get", http.MethodGet, adminKeywordsPath + "/" + id},
		{"Update", http.MethodPut, adminKeywordsPath + "/" + id},
		{"Delete", http.MethodDelete, adminKeywordsPath + "/" + id},
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
				// A mock without funcs panics when called, so a request that
				// is not rejected by the middleware fails loudly.
				repo := &mockrepo.KeywordRepoerMock{}
				app := newAdminKeywordTestApp(t, repo)

				rec := serveAdminKeyword(app, rt.method, rt.path, c.authorization, body)
				expectKeywordStatus(t, rec, c.want)

				if n := keywordRepoCallCount(repo); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	}
}

func TestListKeywordsHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	newRepo := func(
		gotQuery *data.SupportsPagination,
		gotFilter *data.AdminListFilter,
	) *mockrepo.KeywordRepoerMock {
		return &mockrepo.KeywordRepoerMock{
			PaginateFunc: func(
				_ context.Context,
				q data.SupportsPagination,
				f data.AdminListFilter,
			) (*dto.PaginationResults[model.DiscoveryKeyword], error) {
				*gotQuery, *gotFilter = q, f
				return &dto.PaginationResults[model.DiscoveryKeyword]{
					Page:       2,
					PerPage:    5,
					TotalItems: 6,
					Items:      []model.DiscoveryKeyword{*keywordFixture(uuid.New())},
				}, nil
			},
		}
	}

	t.Run("Success_PassesPagingAndFilters", func(t *testing.T) {
		var gotQuery data.SupportsPagination
		var gotFilter data.AdminListFilter
		app := newAdminKeywordTestApp(t, newRepo(&gotQuery, &gotFilter))

		rec := serveAdminKeyword(
			app, http.MethodGet,
			adminKeywordsPath+"?page=2&per_page=5&is_active=true&search=vac", admin, nil,
		)
		expectKeywordStatus(t, rec, http.StatusOK)

		got := decodeKeywordData[dto.PaginationResults[model.DiscoveryKeyword]](t, rec)
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
		if gotFilter.IsActive == nil || !*gotFilter.IsActive || gotFilter.Search != "vac" {
			t.Errorf("unexpected filter: %+v", gotFilter)
		}
	})

	t.Run("Success_InactiveFilter", func(t *testing.T) {
		var gotQuery data.SupportsPagination
		var gotFilter data.AdminListFilter
		app := newAdminKeywordTestApp(t, newRepo(&gotQuery, &gotFilter))

		rec := serveAdminKeyword(
			app, http.MethodGet, adminKeywordsPath+"?is_active=false", admin, nil,
		)
		expectKeywordStatus(t, rec, http.StatusOK)

		if gotFilter.IsActive == nil || *gotFilter.IsActive {
			t.Errorf("expected is_active=false to reach the repo, got %+v", gotFilter.IsActive)
		}
	})

	t.Run("Success_NoQuery_UsesDefaults", func(t *testing.T) {
		var gotQuery data.SupportsPagination
		var gotFilter data.AdminListFilter
		app := newAdminKeywordTestApp(t, newRepo(&gotQuery, &gotFilter))

		rec := serveAdminKeyword(app, http.MethodGet, adminKeywordsPath, admin, nil)
		expectKeywordStatus(t, rec, http.StatusOK)

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
				repo := &mockrepo.KeywordRepoerMock{}
				app := newAdminKeywordTestApp(t, repo)

				rec := serveAdminKeyword(
					app,
					http.MethodGet,
					adminKeywordsPath+tt.query,
					admin,
					nil,
				)
				expectKeywordStatus(t, rec, tt.want)

				if n := len(repo.PaginateCalls()); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	})

	for name, tt := range keywordRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.KeywordRepoerMock{
				PaginateFunc: func(
					context.Context, data.SupportsPagination, data.AdminListFilter,
				) (*dto.PaginationResults[model.DiscoveryKeyword], error) {
					return nil, tt.err
				},
			}
			app := newAdminKeywordTestApp(t, repo)

			rec := serveAdminKeyword(app, http.MethodGet, adminKeywordsPath, admin, nil)
			expectKeywordStatus(t, rec, tt.want)
		})
	}
}

func TestGetKeywordHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	t.Run("Success", func(t *testing.T) {
		id := uuid.New()
		var gotID uuid.UUID
		repo := &mockrepo.KeywordRepoerMock{
			GetByIDFunc: func(_ context.Context, id uuid.UUID) (*model.DiscoveryKeyword, error) {
				gotID = id
				return keywordFixture(id), nil
			},
		}
		app := newAdminKeywordTestApp(t, repo)

		rec := serveAdminKeyword(app, http.MethodGet, adminKeywordsPath+"/"+id.String(), admin, nil)
		expectKeywordStatus(t, rec, http.StatusOK)

		got := decodeKeywordData[model.DiscoveryKeyword](t, rec)
		if got.ID != id || got.Keyword != "vaccin" || got.LanguageCode != "fr" || !got.IsActive {
			t.Errorf("unexpected keyword in response: %+v", got)
		}
		if gotID != id {
			t.Errorf("expected the repo to be asked for %s, got %s", id, gotID)
		}
	})

	t.Run("InvalidID", func(t *testing.T) {
		repo := &mockrepo.KeywordRepoerMock{}
		app := newAdminKeywordTestApp(t, repo)

		rec := serveAdminKeyword(app, http.MethodGet, adminKeywordsPath+"/not-a-uuid", admin, nil)
		expectKeywordStatus(t, rec, http.StatusBadRequest)

		if n := len(repo.GetByIDCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	for name, tt := range keywordRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.KeywordRepoerMock{
				GetByIDFunc: func(context.Context, uuid.UUID) (*model.DiscoveryKeyword, error) {
					return nil, tt.err
				},
			}
			app := newAdminKeywordTestApp(t, repo)

			rec := serveAdminKeyword(
				app, http.MethodGet, adminKeywordsPath+"/"+uuid.NewString(), admin, nil,
			)
			expectKeywordStatus(t, rec, tt.want)
		})
	}
}

func TestCreateKeywordHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	newRepo := func(saved **model.DiscoveryKeyword) *mockrepo.KeywordRepoerMock {
		return &mockrepo.KeywordRepoerMock{
			CreateFunc: func(_ context.Context, k *model.DiscoveryKeyword) error {
				*saved = k
				k.ID = uuid.New()
				k.LanguageCode = k.Language.String()
				k.CreatedAt, k.UpdatedAt = keywordCreatedAt, keywordCreatedAt
				return nil
			},
		}
	}

	t.Run("Success_NormalizesAndDefaultsToActive", func(t *testing.T) {
		var saved *model.DiscoveryKeyword
		app := newAdminKeywordTestApp(t, newRepo(&saved))

		rec := serveAdminKeyword(app, http.MethodPost, adminKeywordsPath, admin,
			marshalAdminBody(t, map[string]any{"keyword": "  Vaccin   COVID ", "language": "fr"}))
		expectKeywordStatus(t, rec, http.StatusCreated)

		got := decodeKeywordData[model.DiscoveryKeyword](t, rec)
		if got.ID == uuid.Nil || got.Keyword != "vaccin covid" ||
			got.LanguageCode != "fr" || !got.IsActive {
			t.Errorf("unexpected keyword in response: %+v", got)
		}
	})

	t.Run("Success_ExplicitInactive", func(t *testing.T) {
		var saved *model.DiscoveryKeyword
		app := newAdminKeywordTestApp(t, newRepo(&saved))

		rec := serveAdminKeyword(app, http.MethodPost, adminKeywordsPath, admin,
			marshalAdminBody(t, map[string]any{
				"keyword": "flu", "language": "en", "is_active": false,
			}))
		expectKeywordStatus(t, rec, http.StatusCreated)

		if saved == nil || saved.IsActive || saved.Language != model.LanguageEnglish {
			t.Errorf("expected an inactive English keyword to reach the repo, got %+v", saved)
		}
	})

	t.Run("ValidationFailed", func(t *testing.T) {
		bodies := map[string]map[string]any{
			"Keyword_Missing":  {"language": "fr"},
			"Keyword_Blank":    {"keyword": "   ", "language": "fr"},
			"Keyword_TooShort": {"keyword": "a", "language": "fr"},
			"Keyword_TooLong":  {"keyword": strings.Repeat("a", 101), "language": "fr"},
			"Language_Missing": {"keyword": "vaccin"},
			"Language_Unknown": {"keyword": "vaccin", "language": "de"},
		}
		for name, body := range bodies {
			t.Run(name, func(t *testing.T) {
				repo := &mockrepo.KeywordRepoerMock{}
				app := newAdminKeywordTestApp(t, repo)

				rec := serveAdminKeyword(
					app, http.MethodPost, adminKeywordsPath, admin, marshalAdminBody(t, body),
				)
				expectKeywordStatus(t, rec, http.StatusUnprocessableEntity)

				if n := len(repo.CreateCalls()); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		app := newAdminKeywordTestApp(t, &mockrepo.KeywordRepoerMock{})

		rec := serveAdminKeyword(app, http.MethodPost, adminKeywordsPath, admin, []byte(`{`))
		expectKeywordStatus(t, rec, http.StatusBadRequest)
	})

	for name, tt := range keywordRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.KeywordRepoerMock{
				CreateFunc: func(context.Context, *model.DiscoveryKeyword) error { return tt.err },
			}
			app := newAdminKeywordTestApp(t, repo)

			rec := serveAdminKeyword(app, http.MethodPost, adminKeywordsPath, admin,
				marshalAdminBody(t, map[string]any{"keyword": "vaccin", "language": "fr"}))
			expectKeywordStatus(t, rec, tt.want)
		})
	}
}

func TestUpdateKeywordHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)
	validBody := map[string]any{"keyword": "Grippe A", "language": "en", "is_active": false}

	t.Run("Success_ReplacesFields", func(t *testing.T) {
		id := uuid.New()
		var saved *model.DiscoveryKeyword
		repo := &mockrepo.KeywordRepoerMock{
			UpdateFunc: func(_ context.Context, k *model.DiscoveryKeyword) error {
				saved = k
				k.LanguageCode = k.Language.String()
				return nil
			},
		}
		app := newAdminKeywordTestApp(t, repo)

		rec := serveAdminKeyword(app, http.MethodPut, adminKeywordsPath+"/"+id.String(), admin,
			marshalAdminBody(t, validBody))
		expectKeywordStatus(t, rec, http.StatusOK)

		got := decodeKeywordData[model.DiscoveryKeyword](t, rec)
		if got.ID != id || got.Keyword != "grippe a" || got.LanguageCode != "en" || got.IsActive {
			t.Errorf("unexpected keyword in response: %+v", got)
		}
		if saved == nil || saved.ID != id {
			t.Errorf("expected the path ID to reach the repo, got %+v", saved)
		}
	})

	t.Run("InvalidID", func(t *testing.T) {
		repo := &mockrepo.KeywordRepoerMock{}
		app := newAdminKeywordTestApp(t, repo)

		rec := serveAdminKeyword(app, http.MethodPut, adminKeywordsPath+"/not-a-uuid", admin,
			marshalAdminBody(t, validBody))
		expectKeywordStatus(t, rec, http.StatusBadRequest)

		if n := len(repo.UpdateCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	t.Run("ValidationFailed", func(t *testing.T) {
		bodies := map[string]map[string]any{
			"IsActive_Missing": {"keyword": "vaccin", "language": "fr"},
			"Keyword_Blank":    {"keyword": "   ", "language": "fr", "is_active": true},
			"Keyword_TooShort": {"keyword": "a", "language": "fr", "is_active": true},
			"Language_Unknown": {"keyword": "vaccin", "language": "de", "is_active": true},
		}
		for name, body := range bodies {
			t.Run(name, func(t *testing.T) {
				repo := &mockrepo.KeywordRepoerMock{}
				app := newAdminKeywordTestApp(t, repo)

				rec := serveAdminKeyword(app, http.MethodPut,
					adminKeywordsPath+"/"+uuid.NewString(), admin, marshalAdminBody(t, body))
				expectKeywordStatus(t, rec, http.StatusUnprocessableEntity)

				if n := len(repo.UpdateCalls()); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		app := newAdminKeywordTestApp(t, &mockrepo.KeywordRepoerMock{})

		rec := serveAdminKeyword(app, http.MethodPut,
			adminKeywordsPath+"/"+uuid.NewString(), admin, []byte(`{`))
		expectKeywordStatus(t, rec, http.StatusBadRequest)
	})

	for name, tt := range keywordRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.KeywordRepoerMock{
				UpdateFunc: func(context.Context, *model.DiscoveryKeyword) error { return tt.err },
			}
			app := newAdminKeywordTestApp(t, repo)

			rec := serveAdminKeyword(app, http.MethodPut,
				adminKeywordsPath+"/"+uuid.NewString(), admin, marshalAdminBody(t, validBody))
			expectKeywordStatus(t, rec, tt.want)
		})
	}
}

func TestDeleteKeywordHandler(t *testing.T) {
	admin := adminBearerFor(t, uuid.New(), model.RoleAdmin)

	t.Run("Success_NoContent", func(t *testing.T) {
		id := uuid.New()
		var gotID uuid.UUID
		repo := &mockrepo.KeywordRepoerMock{
			DeleteFunc: func(_ context.Context, id uuid.UUID) error {
				gotID = id
				return nil
			},
		}
		app := newAdminKeywordTestApp(t, repo)

		rec := serveAdminKeyword(
			app,
			http.MethodDelete,
			adminKeywordsPath+"/"+id.String(),
			admin,
			nil,
		)
		expectKeywordStatus(t, rec, http.StatusNoContent)

		if rec.Body.Len() != 0 {
			t.Errorf("expected an empty body, got %q", rec.Body.String())
		}
		if gotID != id {
			t.Errorf("expected keyword %s to be deleted, got %s", id, gotID)
		}
	})

	t.Run("InvalidID", func(t *testing.T) {
		repo := &mockrepo.KeywordRepoerMock{}
		app := newAdminKeywordTestApp(t, repo)

		rec := serveAdminKeyword(
			app,
			http.MethodDelete,
			adminKeywordsPath+"/not-a-uuid",
			admin,
			nil,
		)
		expectKeywordStatus(t, rec, http.StatusBadRequest)

		if n := len(repo.DeleteCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	for name, tt := range keywordRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.KeywordRepoerMock{
				DeleteFunc: func(context.Context, uuid.UUID) error { return tt.err },
			}
			app := newAdminKeywordTestApp(t, repo)

			rec := serveAdminKeyword(
				app, http.MethodDelete, adminKeywordsPath+"/"+uuid.NewString(), admin, nil,
			)
			expectKeywordStatus(t, rec, tt.want)
		})
	}
}
