package cmd

import (
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/riverqueue/river"
)

// buildApp initializes all API-facing repositories and returns a configured core.App.
func buildApp(
	config *internal.Config,
	riverClient *river.Client[pgx.Tx],
	pool *pgxpool.Pool,
) *core.App {
	queries := db.New(pool)
	apifyRepo := repo.NewApifyRepo(riverClient)
	videoRepo := repo.NewVideoRepo(
		queries,
		pool,
		riverClient,
	)

	return core.NewApp(
		echo.New(),
		config,
		core.WithApifyRepo(apifyRepo),
		core.WithVideoRepo(videoRepo),
	)
}
