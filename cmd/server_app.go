package cmd

import (
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/riverqueue/river"
)

// buildApp initializes all API-facing repositories and returns a configured core.App.
// mediaProvider is constructed by the caller (cmd/server.go), shared with any
// River workers that also need it, rather than built again here.
func buildApp(
	config *internal.Config,
	pool *pgxpool.Pool,
	riverClient *river.Client[pgx.Tx],
	mediaProvider data.MediaUploadProvider,
) *core.App {
	queries := db.New(pool)

	apifyRepo := repo.NewApifyRepo(riverClient)
	videoRepo := repo.NewVideoRepo(
		queries,
		pool,
		riverClient,
	)
	documentRepo := repo.NewDocumentRepo(db.New(pool), pool, riverClient)

	return core.NewApp(
		echo.New(),
		config,
		core.WithApifyRepo(apifyRepo),
		core.WithVideoRepo(videoRepo),
		core.WithDocumentRepo(documentRepo),
		core.WithMediaProvider(mediaProvider),
	)
}
