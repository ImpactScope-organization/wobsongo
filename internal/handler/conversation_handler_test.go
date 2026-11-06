package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/labstack/echo/v4"
)

const (
	conversationStatsPath  = "/api/v1/conversations/stats"
	conversationVolumePath = "/api/v1/conversations/volume"
)

// newConversationTestApp builds the app without signing requests in, so tests
// can choose between an authenticated and an anonymous caller.
func newConversationTestApp(t *testing.T, repo data.ConversationRepoer) *echo.Echo {
	t.Helper()
	// The agent is enabled by default and then needs an LLM endpoint to be
	// built. It is not under test here.
	t.Setenv("AGENT_ENABLED", "false")
	useTestJWTSecret(t)

	app := core.NewApp(
		testhelpers.NewEcho(),
		internal.NewConfig(),
		core.WithConversationRepo(repo),
	)
	return app.Echo()
}

func TestConversationStatsHandler(t *testing.T) {
	t.Run("Success_AllTimeByDefault", func(t *testing.T) {
		var capturedFilter data.UserActivityFilter
		repo := &mockrepo.ConversationRepoerMock{
			UserActivityFunc: func(
				_ context.Context,
				filter data.UserActivityFilter,
			) (*model.UserActivity, error) {
				capturedFilter = filter
				return &model.UserActivity{Users: 59, Messages: 502}, nil
			},
		}
		app := signInAs(t, newConversationTestApp(t, repo), model.RoleUser)

		req := httptest.NewRequest(http.MethodGet, conversationStatsPath, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		if capturedFilter.Days != nil {
			t.Errorf("expected no days filter, got %d", *capturedFilter.Days)
		}

		var resp testhelpers.APIResponse[model.UserActivity]
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}
		if resp.Data.Users != 59 || resp.Data.Messages != 502 {
			t.Errorf("expected 59 users and 502 messages, got %+v", resp.Data)
		}
	})

	t.Run("Days_ReachRepo", func(t *testing.T) {
		for _, days := range []int32{7, 15, 30} {
			var capturedFilter data.UserActivityFilter
			repo := &mockrepo.ConversationRepoerMock{
				UserActivityFunc: func(
					_ context.Context,
					filter data.UserActivityFilter,
				) (*model.UserActivity, error) {
					capturedFilter = filter
					return &model.UserActivity{}, nil
				},
			}
			app := signInAs(t, newConversationTestApp(t, repo), model.RoleUser)

			req := httptest.NewRequest(
				http.MethodGet,
				conversationStatsPath+"?days="+strconv.Itoa(int(days)),
				nil,
			)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("days=%d: expected status %d, got %d", days, http.StatusOK, rec.Code)
			}
			if capturedFilter.Days == nil || *capturedFilter.Days != days {
				t.Errorf("expected days=%d to reach the repo, got %v", days, capturedFilter.Days)
			}
		}
	})

	t.Run("MalformedQuery_DaysNotNumber", func(t *testing.T) {
		app := signInAs(
			t,
			newConversationTestApp(t, &mockrepo.ConversationRepoerMock{}),
			model.RoleUser,
		)

		req := httptest.NewRequest(http.MethodGet, conversationStatsPath+"?days=abc", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("ValidationFailure_UnsupportedDays", func(t *testing.T) {
		app := signInAs(
			t,
			newConversationTestApp(t, &mockrepo.ConversationRepoerMock{}),
			model.RoleUser,
		)

		req := httptest.NewRequest(http.MethodGet, conversationStatsPath+"?days=10", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf(
				"expected status %d, got %d: %s",
				http.StatusUnprocessableEntity,
				rec.Code,
				rec.Body.String(),
			)
		}
	})

	t.Run("RepoInternalError", func(t *testing.T) {
		repo := &mockrepo.ConversationRepoerMock{
			UserActivityFunc: func(
				context.Context,
				data.UserActivityFilter,
			) (*model.UserActivity, error) {
				return nil, data.ErrInternal
			},
		}
		app := signInAs(t, newConversationTestApp(t, repo), model.RoleUser)

		req := httptest.NewRequest(http.MethodGet, conversationStatsPath, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf(
				"expected status %d, got %d: %s",
				http.StatusInternalServerError,
				rec.Code,
				rec.Body.String(),
			)
		}
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		app := newConversationTestApp(t, &mockrepo.ConversationRepoerMock{})

		req := httptest.NewRequest(http.MethodGet, conversationStatsPath, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf(
				"expected status %d, got %d: %s",
				http.StatusUnauthorized,
				rec.Code,
				rec.Body.String(),
			)
		}
	})
}

func TestConversationVolumeHandler(t *testing.T) {
	const day = 24 * time.Hour

	t.Run("Success_24hByDefault", func(t *testing.T) {
		var capturedFilter data.MessageVolumeFilter
		repo := &mockrepo.ConversationRepoerMock{
			MessageVolumeFunc: func(
				_ context.Context,
				filter data.MessageVolumeFilter,
			) (*model.MessageVolume, error) {
				capturedFilter = filter
				return &model.MessageVolume{Current: 12, Previous: 8, Trend: []int64{5, 7}}, nil
			},
		}
		app := signInAs(t, newConversationTestApp(t, repo), model.RoleUser)

		req := httptest.NewRequest(http.MethodGet, conversationVolumePath, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		if capturedFilter.Window != day || capturedFilter.Bucket != time.Hour {
			t.Errorf("expected a 24h window of hourly buckets, got %+v", capturedFilter)
		}

		var resp testhelpers.APIResponse[model.MessageVolume]
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}
		if resp.Data.Current != 12 || resp.Data.Previous != 8 || len(resp.Data.Trend) != 2 {
			t.Errorf("expected 12 current, 8 previous and 2 buckets, got %+v", resp.Data)
		}
	})

	t.Run("Range_ReachesRepo", func(t *testing.T) {
		cases := []struct {
			query  string
			window time.Duration
			bucket time.Duration
		}{
			{"24h", day, time.Hour},
			{"7d", 7 * day, day},
			{"30d", 30 * day, day},
		}
		for _, tc := range cases {
			var capturedFilter data.MessageVolumeFilter
			repo := &mockrepo.ConversationRepoerMock{
				MessageVolumeFunc: func(
					_ context.Context,
					filter data.MessageVolumeFilter,
				) (*model.MessageVolume, error) {
					capturedFilter = filter
					return &model.MessageVolume{}, nil
				},
			}
			app := signInAs(t, newConversationTestApp(t, repo), model.RoleUser)

			req := httptest.NewRequest(
				http.MethodGet,
				conversationVolumePath+"?range="+tc.query,
				nil,
			)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("range=%s: expected status %d, got %d", tc.query, http.StatusOK, rec.Code)
			}
			if capturedFilter.Window != tc.window || capturedFilter.Bucket != tc.bucket {
				t.Errorf("range=%s: expected %s/%s, got %+v",
					tc.query, tc.window, tc.bucket, capturedFilter)
			}
		}
	})

	t.Run("ValidationFailure_UnsupportedRange", func(t *testing.T) {
		app := signInAs(
			t,
			newConversationTestApp(t, &mockrepo.ConversationRepoerMock{}),
			model.RoleUser,
		)

		req := httptest.NewRequest(http.MethodGet, conversationVolumePath+"?range=15d", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf(
				"expected status %d, got %d: %s",
				http.StatusUnprocessableEntity,
				rec.Code,
				rec.Body.String(),
			)
		}
	})

	t.Run("RepoInternalError", func(t *testing.T) {
		repo := &mockrepo.ConversationRepoerMock{
			MessageVolumeFunc: func(
				context.Context,
				data.MessageVolumeFilter,
			) (*model.MessageVolume, error) {
				return nil, data.ErrInternal
			},
		}
		app := signInAs(t, newConversationTestApp(t, repo), model.RoleUser)

		req := httptest.NewRequest(http.MethodGet, conversationVolumePath, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf(
				"expected status %d, got %d: %s",
				http.StatusInternalServerError,
				rec.Code,
				rec.Body.String(),
			)
		}
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		app := newConversationTestApp(t, &mockrepo.ConversationRepoerMock{})

		req := httptest.NewRequest(http.MethodGet, conversationVolumePath, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf(
				"expected status %d, got %d: %s",
				http.StatusUnauthorized,
				rec.Code,
				rec.Body.String(),
			)
		}
	})
}
