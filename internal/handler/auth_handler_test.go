package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/labstack/echo/v4"
)

const testPassword = "correct-horse-battery"

// testUsers is an in-memory UserRepoer seeded with one user per role.
type testUsers struct {
	mock  *mockrepo.UserRepoerMock
	byID  map[uuid.UUID]*model.User
	alice *model.User
}

func newTestUsers(t *testing.T) *testUsers {
	t.Helper()
	hash, err := auth.HashPassword(testPassword, auth.WithMinCost())
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}
	u := &testUsers{byID: map[uuid.UUID]*model.User{}}
	u.alice = &model.User{
		ID:                uuid.New(),
		Name:              "Alice",
		Email:             "alice@example.com",
		PasswordHash:      hash,
		Role:              model.RoleUser,
		PasswordChangedAt: time.Now().Add(-time.Hour),
	}
	u.byID[u.alice.ID] = u.alice
	byEmail := func(email string) *model.User {
		for _, user := range u.byID {
			if user.Email == email {
				return user
			}
		}
		return nil
	}
	u.mock = &mockrepo.UserRepoerMock{
		GetByIDFunc: func(_ context.Context, id uuid.UUID) (*model.User, error) {
			if user, ok := u.byID[id]; ok {
				return user, nil
			}
			return nil, data.ErrNotFound
		},
		GetByEmailFunc: func(_ context.Context, email string) (*model.User, error) {
			if user := byEmail(email); user != nil {
				return user, nil
			}
			return nil, data.ErrNotFound
		},
		CreateFunc: func(_ context.Context, user *model.User) error {
			if byEmail(user.Email) != nil {
				return data.ErrConflict
			}
			user.ID = uuid.New()
			user.PasswordChangedAt = time.Now()
			u.byID[user.ID] = user
			return nil
		},
		UpdateProfileFunc: func(_ context.Context, id uuid.UUID, name, email string) (*model.User, error) {
			if other := byEmail(email); other != nil && other.ID != id {
				return nil, data.ErrConflict
			}
			user := u.byID[id]
			user.Name, user.Email = name, email
			return user, nil
		},
		UpdatePasswordFunc: func(_ context.Context, id uuid.UUID, hash string) (*model.User, error) {
			user := u.byID[id]
			user.PasswordHash = hash
			user.PasswordChangedAt = time.Now()
			return user, nil
		},
	}
	return u
}

func newAuthTestApp(t *testing.T, users *testUsers) *echo.Echo {
	t.Helper()
	t.Setenv("AGENT_ENABLED", "false")
	useTestJWTSecret(t)
	app := core.NewApp(
		testhelpers.NewEcho(), internal.NewConfig(), core.WithUserRepo(users.mock),
	)
	return app.Echo()
}

type apiResult struct {
	code  int
	data  map[string]any
	error string
}

func call(t *testing.T, app *echo.Echo, method, path, bearer string, body any) apiResult {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if bearer != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var envelope struct {
		Data  map[string]any `json:"data"`
		Error string         `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &envelope)
	return apiResult{code: rec.Code, data: envelope.Data, error: envelope.Error}
}

func expectStatus(t *testing.T, got apiResult, want int) {
	t.Helper()
	if got.code != want {
		t.Fatalf("expected status %d, got %d (%s)", want, got.code, got.error)
	}
}

func login(t *testing.T, app *echo.Echo, email, password string) apiResult {
	t.Helper()
	return call(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": email, "password": password,
	})
}

func TestAuthLogin(t *testing.T) {
	app := newAuthTestApp(t, newTestUsers(t))

	t.Run("returns a token pair, whatever the email case", func(t *testing.T) {
		res := login(t, app, " ALICE@example.com", testPassword)
		expectStatus(t, res, http.StatusOK)
		for _, key := range []string{
			"access_token", "access_expires_at", "refresh_token", "refresh_expires_at",
		} {
			if res.data[key] == nil || res.data[key] == "" {
				t.Errorf("missing %s", key)
			}
		}
	})

	t.Run("gives the same 401 for a wrong password and an unknown email", func(t *testing.T) {
		wrong := login(t, app, "alice@example.com", "wrong-password")
		unknown := login(t, app, "nobody@example.com", testPassword)
		expectStatus(t, wrong, http.StatusUnauthorized)
		expectStatus(t, unknown, http.StatusUnauthorized)
		if wrong.error != unknown.error {
			t.Errorf("messages differ: %q vs %q", wrong.error, unknown.error)
		}
	})

	t.Run("requires both fields", func(t *testing.T) {
		expectStatus(t, login(t, app, "", ""), http.StatusUnprocessableEntity)
	})
}

func TestAuthSignup(t *testing.T) {
	signup := func(app *echo.Echo, email, password string) apiResult {
		return call(t, app, http.MethodPost, "/api/v1/auth/signup", "", map[string]string{
			"name": "Bob", "email": email, "password": password,
		})
	}

	t.Run("creates a user-role account, signed in", func(t *testing.T) {
		app := newAuthTestApp(t, newTestUsers(t))
		res := signup(app, "bob@example.com", "bob-password-12")
		expectStatus(t, res, http.StatusCreated)

		me := call(t, app, http.MethodGet, "/api/v1/me", res.data["access_token"].(string), nil)
		expectStatus(t, me, http.StatusOK)
		if me.data["role"] != "user" || me.data["email"] != "bob@example.com" {
			t.Errorf("unexpected account: %v", me.data)
		}
	})

	t.Run("rejects a taken email, whatever the case", func(t *testing.T) {
		app := newAuthTestApp(t, newTestUsers(t))
		expectStatus(t, signup(app, "Alice@Example.com", "bob-password-12"), http.StatusConflict)
	})

	t.Run("enforces the password policy", func(t *testing.T) {
		app := newAuthTestApp(t, newTestUsers(t))
		res := signup(app, "bob@example.com", "short")
		expectStatus(t, res, http.StatusUnprocessableEntity)
		if res.error == "" {
			t.Error("expected a message explaining the policy")
		}
	})

	t.Run("rejects an invalid email", func(t *testing.T) {
		app := newAuthTestApp(t, newTestUsers(t))
		expectStatus(
			t,
			signup(app, "not-an-email", "bob-password-12"),
			http.StatusUnprocessableEntity,
		)
	})
}

func TestAuthRefresh(t *testing.T) {
	app := newAuthTestApp(t, newTestUsers(t))
	tokens := login(t, app, "alice@example.com", testPassword).data
	refresh := func(token any) apiResult {
		return call(t, app, http.MethodPost, "/api/v1/auth/refresh", "", map[string]any{
			"refresh_token": token,
		})
	}

	expectStatus(t, refresh(tokens["refresh_token"]), http.StatusOK)
	expectStatus(t, refresh(tokens["access_token"]), http.StatusUnauthorized)
	expectStatus(t, refresh("garbage"), http.StatusUnauthorized)
}

func TestMe(t *testing.T) {
	users := newTestUsers(t)
	app := newAuthTestApp(t, users)
	access := login(t, app, "alice@example.com", testPassword).data["access_token"].(string)

	t.Run("needs a bearer token", func(t *testing.T) {
		expectStatus(
			t,
			call(t, app, http.MethodGet, "/api/v1/me", "", nil),
			http.StatusUnauthorized,
		)
		expectStatus(
			t,
			call(t, app, http.MethodGet, "/api/v1/me", "garbage", nil),
			http.StatusUnauthorized,
		)
	})

	t.Run("returns the account", func(t *testing.T) {
		res := call(t, app, http.MethodGet, "/api/v1/me", access, nil)
		expectStatus(t, res, http.StatusOK)
		if res.data["id"] != users.alice.ID.String() || res.data["name"] != "Alice" {
			t.Errorf("unexpected account: %v", res.data)
		}
		if _, leaked := res.data["password_hash"]; leaked {
			t.Error("password hash must never be returned")
		}
	})

	t.Run("updates the profile", func(t *testing.T) {
		res := call(t, app, http.MethodPatch, "/api/v1/me", access, map[string]string{
			"name": "Alice B.", "email": "Alice.B@Example.com",
		})
		expectStatus(t, res, http.StatusOK)
		if res.data["email"] != "alice.b@example.com" {
			t.Errorf("expected normalized email, got %v", res.data["email"])
		}
	})

	t.Run("rejects another account's email", func(t *testing.T) {
		other := &model.User{ID: uuid.New(), Email: "carol@example.com"}
		users.byID[other.ID] = other
		res := call(t, app, http.MethodPatch, "/api/v1/me", access, map[string]string{
			"name": "Alice", "email": "carol@example.com",
		})
		expectStatus(t, res, http.StatusConflict)
	})
}

func TestChangePassword(t *testing.T) {
	users := newTestUsers(t)
	app := newAuthTestApp(t, users)
	old := login(t, app, "alice@example.com", testPassword).data
	access := old["access_token"].(string)
	change := func(current, next string) apiResult {
		return call(t, app, http.MethodPost, "/api/v1/me/password", access, map[string]string{
			"current_password": current, "new_password": next,
		})
	}

	expectStatus(t, change("wrong-password", "new-password-12"), http.StatusForbidden)
	expectStatus(t, change(testPassword, "short"), http.StatusUnprocessableEntity)

	// Make the old refresh token clearly older than the change.
	users.alice.PasswordChangedAt = time.Now().Add(-time.Hour)
	res := change(testPassword, "new-password-12")
	expectStatus(t, res, http.StatusOK)

	refresh := func(token any) apiResult {
		return call(t, app, http.MethodPost, "/api/v1/auth/refresh", "", map[string]any{
			"refresh_token": token,
		})
	}
	expectStatus(t, refresh(res.data["refresh_token"]), http.StatusOK)
	expectStatus(t, login(t, app, "alice@example.com", "new-password-12"), http.StatusOK)
}

func TestProtectedRoutes(t *testing.T) {
	app := newAuthTestApp(t, newTestUsers(t))
	user := bearerFor(t, model.RoleUser)[len("Bearer "):]

	t.Run("documents need a token", func(t *testing.T) {
		res := call(t, app, http.MethodGet, "/api/v1/documents", "", nil)
		expectStatus(t, res, http.StatusUnauthorized)
	})

	t.Run("documents need the admin role", func(t *testing.T) {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			res := call(t, app, method, "/api/v1/documents", user, nil)
			expectStatus(t, res, http.StatusForbidden)
		}
		res := call(t, app, http.MethodDelete, "/api/v1/documents/"+uuid.NewString(), user, nil)
		expectStatus(t, res, http.StatusForbidden)
	})

	t.Run("claim checks need a token", func(t *testing.T) {
		res := call(
			t,
			app,
			http.MethodPost,
			"/api/v1/claims/check",
			"",
			map[string]string{"text": "x"},
		)
		expectStatus(t, res, http.StatusUnauthorized)
	})

	t.Run("expired tokens are refused", func(t *testing.T) {
		past := testAuth().WithClock(func() time.Time { return time.Now().Add(-time.Hour) })
		tokens, err := past.GenerateJWTPair(&auth.JWTPayload{ID: uuid.New(), Role: model.RoleAdmin})
		if err != nil {
			t.Fatalf("failed to sign: %v", err)
		}
		res := call(t, app, http.MethodGet, "/api/v1/documents", tokens.AccessToken, nil)
		expectStatus(t, res, http.StatusUnauthorized)
	})
}
