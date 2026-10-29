package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/impactscope-organization/wobsongo/internal/utils"
	"github.com/labstack/echo/v4"
)

const narrativesBasePath = "/api/v1/narratives"

// newNarrativeTestApp builds the app without signing requests in, so tests can
// choose between an authenticated and an anonymous caller.
func newNarrativeTestApp(t *testing.T, repo data.NarrativeRepoer) *echo.Echo {
	t.Helper()
	// The agent is enabled by default and then needs an LLM endpoint to be
	// built. It is not under test here.
	t.Setenv("AGENT_ENABLED", "false")
	useTestJWTSecret(t)

	app := core.NewApp(testhelpers.NewEcho(), internal.NewConfig(), core.WithNarrativeRepo(repo))
	return app.Echo()
}

func TestListNarrativesHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var capturedQuery data.SupportsPagination
		var capturedFilter data.NarrativeFilter
		repo := &mockrepo.NarrativeRepoerMock{
			PaginateNarrativesFunc: func(
				_ context.Context,
				q data.SupportsPagination,
				filter data.NarrativeFilter,
			) (*dto.PaginationResults[model.Narrative], error) {
				capturedQuery = q
				capturedFilter = filter
				return &dto.PaginationResults[model.Narrative]{
					Page:       2,
					PerPage:    5,
					TotalItems: 6,
					Items: []model.Narrative{{
						ID:        uuid.New(),
						Statement: "A narrative",
						TopicName: "Family planning",
						Verdict:   utils.PtrOf("contradicted"),
					}},
				}, nil
			},
		}
		app := signInAs(t, newNarrativeTestApp(t, repo), model.RoleUser)

		req := httptest.NewRequest(http.MethodGet, narrativesBasePath+"?page=2&per_page=5", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		if capturedQuery == nil || capturedQuery.Limit() != 5 || capturedQuery.Offset() != 5 {
			t.Errorf("expected limit 5 and offset 5 to reach the repo, got %v", capturedQuery)
		}

		var resp testhelpers.APIResponse[dto.PaginationResults[model.Narrative]]
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}
		if len(resp.Data.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(resp.Data.Items))
		}
		if resp.Data.Items[0].Statement != "A narrative" {
			t.Errorf("expected statement %q, got %q", "A narrative", resp.Data.Items[0].Statement)
		}
		if resp.Data.TotalItems != 6 {
			t.Errorf("expected total items 6, got %d", resp.Data.TotalItems)
		}
		if capturedFilter.IsActive != nil || capturedFilter.Status != "" {
			t.Errorf("expected no filter, got %+v", capturedFilter)
		}
		item := resp.Data.Items[0]
		if item.TopicName != "Family planning" {
			t.Errorf("expected topic_name %q, got %q", "Family planning", item.TopicName)
		}
		if item.Verdict == nil || *item.Verdict != "contradicted" {
			t.Errorf("expected verdict %q, got %v", "contradicted", item.Verdict)
		}
	})

	t.Run("MalformedQuery", func(t *testing.T) {
		app := signInAs(t, newNarrativeTestApp(t, &mockrepo.NarrativeRepoerMock{}), model.RoleUser)

		req := httptest.NewRequest(http.MethodGet, narrativesBasePath+"?page=abc", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf(
				"expected status %d, got %d: %s",
				http.StatusBadRequest,
				rec.Code,
				rec.Body.String(),
			)
		}
	})

	t.Run("Filters_ReachRepo", func(t *testing.T) {
		var capturedFilter data.NarrativeFilter
		repo := &mockrepo.NarrativeRepoerMock{
			PaginateNarrativesFunc: func(
				_ context.Context,
				_ data.SupportsPagination,
				filter data.NarrativeFilter,
			) (*dto.PaginationResults[model.Narrative], error) {
				capturedFilter = filter
				return &dto.PaginationResults[model.Narrative]{}, nil
			},
		}
		app := signInAs(t, newNarrativeTestApp(t, repo), model.RoleUser)

		req := httptest.NewRequest(
			http.MethodGet,
			narrativesBasePath+"?is_active=false&status=expert_verified",
			nil,
		)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		if capturedFilter.IsActive == nil || *capturedFilter.IsActive {
			t.Errorf("expected is_active=false to reach the repo, got %v", capturedFilter.IsActive)
		}
		if capturedFilter.Status != "expert_verified" {
			t.Errorf("expected status %q, got %q", "expert_verified", capturedFilter.Status)
		}
	})

	t.Run("ValidationFailure_UnknownStatus", func(t *testing.T) {
		app := signInAs(t, newNarrativeTestApp(t, &mockrepo.NarrativeRepoerMock{}), model.RoleUser)

		req := httptest.NewRequest(http.MethodGet, narrativesBasePath+"?status=bogus", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
		}
	})

	t.Run("MalformedQuery_IsActiveNotBool", func(t *testing.T) {
		app := signInAs(t, newNarrativeTestApp(t, &mockrepo.NarrativeRepoerMock{}), model.RoleUser)

		req := httptest.NewRequest(http.MethodGet, narrativesBasePath+"?is_active=maybe", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("ValidationFailure_PerPageTooLarge", func(t *testing.T) {
		app := signInAs(t, newNarrativeTestApp(t, &mockrepo.NarrativeRepoerMock{}), model.RoleUser)

		req := httptest.NewRequest(http.MethodGet, narrativesBasePath+"?per_page=101", nil)
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
		repo := &mockrepo.NarrativeRepoerMock{
			PaginateNarrativesFunc: func(
				context.Context,
				data.SupportsPagination,
				data.NarrativeFilter,
			) (*dto.PaginationResults[model.Narrative], error) {
				return nil, errors.New("internal error")
			},
		}
		app := signInAs(t, newNarrativeTestApp(t, repo), model.RoleUser)

		req := httptest.NewRequest(http.MethodGet, narrativesBasePath, nil)
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
		app := newNarrativeTestApp(t, &mockrepo.NarrativeRepoerMock{})

		req := httptest.NewRequest(http.MethodGet, narrativesBasePath, nil)
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

func TestGetNarrativeHandler(t *testing.T) {
	repo := &mockrepo.NarrativeRepoerMock{
		GetNarrativeFunc: func(_ context.Context, id uuid.UUID) (*model.Narrative, error) {
			if err := testhelpers.ErrorForUUID(id); err != nil {
				return nil, err
			}
			return &model.Narrative{
				ID:        id,
				Statement: "Found Narrative",
				Status:    model.NarrativeStatusNeedsCheck,
			}, nil
		},
	}
	app := signInAs(t, newNarrativeTestApp(t, repo), model.RoleUser)

	cases := []struct {
		name           string
		id             string
		expectedStatus int
	}{
		{"Success", testhelpers.NewUUIDWithSuffix(testhelpers.SuffixOK).String(), http.StatusOK},
		{
			"NotFound",
			testhelpers.NewUUIDWithSuffix(testhelpers.SuffixNotFound).String(),
			http.StatusNotFound,
		},
		{
			"Forbidden",
			testhelpers.NewUUIDWithSuffix(testhelpers.SuffixForbidden).String(),
			http.StatusForbidden,
		},
		{
			"InternalError",
			testhelpers.NewUUIDWithSuffix(testhelpers.SuffixInternal).String(),
			http.StatusInternalServerError,
		},
		{"InvalidUUID", "not-a-uuid", http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, narrativesBasePath+"/"+tc.id, nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf(
					"expected status %d, got %d: %s",
					tc.expectedStatus,
					rec.Code,
					rec.Body.String(),
				)
			}
			if tc.expectedStatus == http.StatusOK {
				var resp testhelpers.APIResponse[model.Narrative]
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response body: %v", err)
				}
				if resp.Data.ID.String() != tc.id {
					t.Errorf("expected narrative ID %s, got %s", tc.id, resp.Data.ID)
				}
				if resp.Data.Statement != "Found Narrative" {
					t.Errorf(
						"expected statement %q, got %q",
						"Found Narrative",
						resp.Data.Statement,
					)
				}
			}
		})
	}

	t.Run("Unauthenticated", func(t *testing.T) {
		anonymous := newNarrativeTestApp(t, repo)
		id := testhelpers.NewUUIDWithSuffix(testhelpers.SuffixOK)

		req := httptest.NewRequest(http.MethodGet, narrativesBasePath+"/"+id.String(), nil)
		rec := httptest.NewRecorder()
		anonymous.ServeHTTP(rec, req)

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
