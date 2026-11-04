package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/labstack/echo/v4"
)

const adminSettingsPath = "/api/v1/admin/settings"

var adminSettingUpdatedAt = time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)

// newAdminSettingTestApp builds the app without signing requests in, so each
// test picks its own caller (admin, plain user, or nobody).
func newAdminSettingTestApp(
	t *testing.T,
	repo data.CrawlerRepoer,
	rescheduler data.DiscoveryRescheduler,
) *echo.Echo {
	t.Helper()
	// The agent is enabled by default and then needs an LLM endpoint to be
	// built. It is not under test here.
	t.Setenv("AGENT_ENABLED", "false")
	useTestJWTSecret(t)

	app := core.NewApp(
		testhelpers.NewEcho(),
		internal.NewConfig(),
		core.WithCrawlerRepo(repo),
		core.WithDiscoveryRescheduler(rescheduler),
	)
	return app.Echo()
}

// adminBearerFor signs an access token for a known user ID, so tests can check
// which user the handler records as the one who changed the settings.
func adminBearerFor(t *testing.T, id uuid.UUID, role model.UserRole) string {
	t.Helper()
	tokens, err := testAuth().GenerateJWTPair(&auth.JWTPayload{
		ID:    id,
		Name:  "Test Admin",
		Email: "admin@example.com",
		Role:  role,
	})
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return "Bearer " + tokens.AccessToken
}

func serveAdminSetting(
	app *echo.Echo,
	method, authorization string,
	body []byte,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, adminSettingsPath, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if authorization != "" {
		req.Header.Set(echo.HeaderAuthorization, authorization)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func marshalAdminBody(t *testing.T, v any) []byte {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	return body
}

func expectAdminSettingStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("expected status %d, got %d: %s", want, rec.Code, rec.Body.String())
	}
}

func validUpdateSettingBody() dto.UpdateAdminSettingRequest {
	return dto.UpdateAdminSettingRequest{
		DiscoveryIntervalHours: 6,
		MaxResultsKeyword:      30,
		MaxResultsInfluencer:   15,
		MaxCostPerRun:          12.5,
	}
}

func TestGetSettingHandler(t *testing.T) {
	adminID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		stored := &model.AdminSetting{
			DiscoveryIntervalHours: 24,
			MaxResultsKeyword:      20,
			MaxResultsInfluencer:   10,
			MaxCostPerRun:          5,
			UpdatedAt:              adminSettingUpdatedAt,
		}
		repo := &mockrepo.CrawlerRepoerMock{
			GetSettingFunc: func(context.Context) (*model.AdminSetting, error) {
				return stored, nil
			},
		}
		app := newAdminSettingTestApp(t, repo, nil)

		rec := serveAdminSetting(
			app, http.MethodGet, adminBearerFor(t, adminID, model.RoleAdmin), nil,
		)
		expectAdminSettingStatus(t, rec, http.StatusOK)

		var resp testhelpers.APIResponse[model.AdminSetting]
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}
		got := resp.Data
		if got.DiscoveryIntervalHours != 24 || got.MaxResultsKeyword != 20 ||
			got.MaxResultsInfluencer != 10 || got.MaxCostPerRun != 5 {
			t.Errorf("unexpected setting in response: %+v", got)
		}
		if !got.UpdatedAt.Equal(adminSettingUpdatedAt) {
			t.Errorf("expected updated_at %v, got %v", adminSettingUpdatedAt, got.UpdatedAt)
		}
	})

	t.Run("Superadmin_IsAllowed", func(t *testing.T) {
		repo := &mockrepo.CrawlerRepoerMock{
			GetSettingFunc: func(context.Context) (*model.AdminSetting, error) {
				return &model.AdminSetting{}, nil
			},
		}
		app := newAdminSettingTestApp(t, repo, nil)

		rec := serveAdminSetting(
			app, http.MethodGet, adminBearerFor(t, adminID, model.RoleSuperadmin), nil,
		)
		expectAdminSettingStatus(t, rec, http.StatusOK)
	})

	for name, repoErr := range map[string]error{
		"RepoError_Internal": fmt.Errorf("read setting: %w", data.ErrInternal),
		"RepoError_Unmapped": errors.New("connection reset"),
	} {
		t.Run(name, func(t *testing.T) {
			repo := &mockrepo.CrawlerRepoerMock{
				GetSettingFunc: func(context.Context) (*model.AdminSetting, error) {
					return nil, repoErr
				},
			}
			app := newAdminSettingTestApp(t, repo, nil)

			rec := serveAdminSetting(
				app, http.MethodGet, adminBearerFor(t, adminID, model.RoleAdmin), nil,
			)
			expectAdminSettingStatus(t, rec, http.StatusInternalServerError)
		})
	}
}

func TestAdminSettingRoutes_Authorization(t *testing.T) {
	body := marshalAdminBody(t, validUpdateSettingBody())

	tests := []struct {
		name          string
		method        string
		authorization string
		wantStatus    int
	}{
		{"Get_WithoutToken", http.MethodGet, "", http.StatusUnauthorized},
		{"Put_WithoutToken", http.MethodPut, "", http.StatusUnauthorized},
		{"Get_InvalidToken", http.MethodGet, "Bearer not-a-token", http.StatusUnauthorized},
		{"Get_AsUser", http.MethodGet, bearerFor(t, model.RoleUser), http.StatusForbidden},
		{"Put_AsUser", http.MethodPut, bearerFor(t, model.RoleUser), http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A mock without funcs panics when called, so a request that is
			// not rejected by the middleware fails loudly.
			repo := &mockrepo.CrawlerRepoerMock{}
			app := newAdminSettingTestApp(t, repo, nil)

			rec := serveAdminSetting(app, tt.method, tt.authorization, body)
			expectAdminSettingStatus(t, rec, tt.wantStatus)

			if n := len(repo.GetSettingCalls()) + len(repo.UpdateSettingCalls()); n != 0 {
				t.Errorf("expected the repo not to be called, got %d calls", n)
			}
		})
	}
}

func TestUpdateSettingHandler(t *testing.T) {
	adminID := uuid.New()

	t.Run("Success_RecordsCallerAndReschedules", func(t *testing.T) {
		repo := &mockrepo.CrawlerRepoerMock{
			UpdateSettingFunc: func(_ context.Context, s *model.AdminSetting) (int, error) {
				s.UpdatedAt = adminSettingUpdatedAt
				return 24, nil // stored interval before this update
			},
		}
		rescheduler := &mockrepo.DiscoveryReschedulerMock{RescheduleFunc: func(int, bool) {}}
		app := newAdminSettingTestApp(t, repo, rescheduler)

		rec := serveAdminSetting(
			app,
			http.MethodPut,
			adminBearerFor(t, adminID, model.RoleAdmin),
			marshalAdminBody(t, validUpdateSettingBody()),
		)
		expectAdminSettingStatus(t, rec, http.StatusOK)

		var resp testhelpers.APIResponse[model.AdminSetting]
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}
		got := resp.Data
		if got.DiscoveryIntervalHours != 6 || got.MaxResultsKeyword != 30 ||
			got.MaxResultsInfluencer != 15 || got.MaxCostPerRun != 12.5 {
			t.Errorf("unexpected setting in response: %+v", got)
		}
		if got.UpdatedBy == nil || *got.UpdatedBy != adminID {
			t.Errorf("expected updated_by %s, got %v", adminID, got.UpdatedBy)
		}

		calls := repo.UpdateSettingCalls()
		if len(calls) != 1 || calls[0].Setting.UpdatedBy == nil ||
			*calls[0].Setting.UpdatedBy != adminID {
			t.Errorf("expected the caller ID to reach the repo, got %+v", calls)
		}

		reschedules := rescheduler.RescheduleCalls()
		if len(reschedules) != 1 || reschedules[0].IntervalHours != 6 ||
			reschedules[0].RunOnStart {
			t.Errorf("expected Reschedule(6, false), got %+v", reschedules)
		}
	})

	t.Run("Success_IntervalUnchanged_DoesNotReschedule", func(t *testing.T) {
		repo := &mockrepo.CrawlerRepoerMock{
			UpdateSettingFunc: func(context.Context, *model.AdminSetting) (int, error) {
				return 6, nil
			},
		}
		rescheduler := &mockrepo.DiscoveryReschedulerMock{RescheduleFunc: func(int, bool) {}}
		app := newAdminSettingTestApp(t, repo, rescheduler)

		rec := serveAdminSetting(
			app,
			http.MethodPut,
			adminBearerFor(t, adminID, model.RoleAdmin),
			marshalAdminBody(t, validUpdateSettingBody()),
		)
		expectAdminSettingStatus(t, rec, http.StatusOK)

		if n := len(rescheduler.RescheduleCalls()); n != 0 {
			t.Errorf("expected no reschedule, got %d", n)
		}
	})

	t.Run("Success_WithoutRescheduler", func(t *testing.T) {
		repo := &mockrepo.CrawlerRepoerMock{
			UpdateSettingFunc: func(context.Context, *model.AdminSetting) (int, error) {
				return 24, nil
			},
		}
		app := newAdminSettingTestApp(t, repo, nil)

		rec := serveAdminSetting(
			app,
			http.MethodPut,
			adminBearerFor(t, adminID, model.RoleAdmin),
			marshalAdminBody(t, validUpdateSettingBody()),
		)
		expectAdminSettingStatus(t, rec, http.StatusOK)
	})

	t.Run("Success_AcceptsBoundaryValues", func(t *testing.T) {
		bodies := map[string]dto.UpdateAdminSettingRequest{
			"Minimum": {
				DiscoveryIntervalHours: 1, MaxResultsKeyword: 1,
				MaxResultsInfluencer: 1, MaxCostPerRun: 0.01,
			},
			"Maximum": {
				DiscoveryIntervalHours: 720, MaxResultsKeyword: 100,
				MaxResultsInfluencer: 100, MaxCostPerRun: 1000,
			},
		}
		for name, body := range bodies {
			t.Run(name, func(t *testing.T) {
				repo := &mockrepo.CrawlerRepoerMock{
					UpdateSettingFunc: func(context.Context, *model.AdminSetting) (int, error) {
						return 0, nil
					},
				}
				app := newAdminSettingTestApp(t, repo, nil)

				rec := serveAdminSetting(
					app,
					http.MethodPut,
					adminBearerFor(t, adminID, model.RoleAdmin),
					marshalAdminBody(t, body),
				)
				expectAdminSettingStatus(t, rec, http.StatusOK)
			})
		}
	})

	t.Run("ValidationFailed", func(t *testing.T) {
		mutations := map[string]func(*dto.UpdateAdminSettingRequest){
			"Interval_Zero":            func(r *dto.UpdateAdminSettingRequest) { r.DiscoveryIntervalHours = 0 },
			"Interval_TooLarge":        func(r *dto.UpdateAdminSettingRequest) { r.DiscoveryIntervalHours = 721 },
			"KeywordResults_Zero":      func(r *dto.UpdateAdminSettingRequest) { r.MaxResultsKeyword = 0 },
			"KeywordResults_TooLarge":  func(r *dto.UpdateAdminSettingRequest) { r.MaxResultsKeyword = 101 },
			"InfluencerResults_Zero":   func(r *dto.UpdateAdminSettingRequest) { r.MaxResultsInfluencer = 0 },
			"InfluencerResults_TooBig": func(r *dto.UpdateAdminSettingRequest) { r.MaxResultsInfluencer = 101 },
			"Cost_Zero":                func(r *dto.UpdateAdminSettingRequest) { r.MaxCostPerRun = 0 },
			"Cost_Negative":            func(r *dto.UpdateAdminSettingRequest) { r.MaxCostPerRun = -1 },
			"Cost_TooLarge":            func(r *dto.UpdateAdminSettingRequest) { r.MaxCostPerRun = 1000.01 },
		}
		for name, mutate := range mutations {
			t.Run(name, func(t *testing.T) {
				repo := &mockrepo.CrawlerRepoerMock{}
				app := newAdminSettingTestApp(t, repo, nil)

				body := validUpdateSettingBody()
				mutate(&body)
				rec := serveAdminSetting(
					app,
					http.MethodPut,
					adminBearerFor(t, adminID, model.RoleAdmin),
					marshalAdminBody(t, body),
				)
				expectAdminSettingStatus(t, rec, http.StatusUnprocessableEntity)

				if n := len(repo.UpdateSettingCalls()); n != 0 {
					t.Errorf("expected the repo not to be called, got %d calls", n)
				}
			})
		}
	})

	t.Run("EmptyObject_Fails", func(t *testing.T) {
		app := newAdminSettingTestApp(t, &mockrepo.CrawlerRepoerMock{}, nil)

		rec := serveAdminSetting(
			app,
			http.MethodPut,
			adminBearerFor(t, adminID, model.RoleAdmin),
			[]byte(`{}`),
		)
		expectAdminSettingStatus(t, rec, http.StatusUnprocessableEntity)
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		app := newAdminSettingTestApp(t, &mockrepo.CrawlerRepoerMock{}, nil)

		rec := serveAdminSetting(
			app,
			http.MethodPut,
			adminBearerFor(t, adminID, model.RoleAdmin),
			[]byte(`{`),
		)
		expectAdminSettingStatus(t, rec, http.StatusBadRequest)
	})

	t.Run("RepoError_DoesNotReschedule", func(t *testing.T) {
		repo := &mockrepo.CrawlerRepoerMock{
			UpdateSettingFunc: func(context.Context, *model.AdminSetting) (int, error) {
				return 0, fmt.Errorf("save setting: %w", data.ErrInternal)
			},
		}
		rescheduler := &mockrepo.DiscoveryReschedulerMock{RescheduleFunc: func(int, bool) {}}
		app := newAdminSettingTestApp(t, repo, rescheduler)

		rec := serveAdminSetting(
			app,
			http.MethodPut,
			adminBearerFor(t, adminID, model.RoleAdmin),
			marshalAdminBody(t, validUpdateSettingBody()),
		)
		expectAdminSettingStatus(t, rec, http.StatusInternalServerError)

		if n := len(rescheduler.RescheduleCalls()); n != 0 {
			t.Errorf("expected no reschedule after a failed update, got %d", n)
		}
	})
}
