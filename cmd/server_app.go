package cmd

import (
	"context"
	"fmt"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/riverqueue/river"
)

// buildApp initializes all API-facing repositories and returns a configured core.App.
func buildApp(
	ctx context.Context,
	config *internal.Config,
	riverClient *river.Client[pgx.Tx],
) (*core.App, error) {
	apifyRepo := repo.NewApifyRepo(riverClient)
	documentRepo := newStubDocumentRepo()

	if err := internal.IsS3OK(config.S3Config); err != nil {
		return nil, fmt.Errorf("S3 configuration required to serve media routes: %w", err)
	}
	mediaProvider, err := repo.NewS3Provider(ctx, config.S3Config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 media provider: %w", err)
	}

	return core.NewApp(
		echo.New(),
		config,
		core.WithApifyRepo(apifyRepo),
		core.WithDocumentRepo(documentRepo),
		core.WithMediaProvider(mediaProvider),
	), nil
}
