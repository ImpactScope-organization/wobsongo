package cmd

import (
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/riverqueue/river"
)

// buildApp initializes all API-facing repositories and returns a configured core.App.
func buildApp(
	config *internal.Config,
	riverClient *river.Client[pgx.Tx],
) *core.App {
	apifyRepo := repo.NewApifyRepo(riverClient)
	documentRepo := newStubDocumentRepo()

	return core.NewApp(
		echo.New(),
		config,
		core.WithApifyRepo(apifyRepo),
		core.WithDocumentRepo(documentRepo),
	)
}
