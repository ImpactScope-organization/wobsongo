package cmd

import (
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/labstack/echo/v4"
)

// buildApp initializes all API-facing repositories and returns a configured core.App.
func buildApp(
	config *internal.Config,
) *core.App {
	return core.NewApp(
		echo.New(),
		config,
	)
}
