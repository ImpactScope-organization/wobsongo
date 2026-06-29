package core

import (
	"fmt"
	"net/http"
	"time"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/handler"
	"github.com/labstack/echo/v4"
)

const (
	ReadTimeout  = 5 * time.Second
	WriteTimeout = 10 * time.Second
)

type App struct {
	config   *internal.Config
	echoApp  *echo.Echo
	apiGroup *echo.Group
}

// Echo returns the Echo instance of the application.
func (app *App) Echo() *echo.Echo {
	return app.echoApp
}

// Config returns the application configuration.
func (app *App) Config() *internal.Config {
	return app.config
}

// Server returns the HTTP server instance configured with the application's Echo instance.
func (app *App) Server() *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.Port),
		Handler:      app.Echo(),
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
	}
}

// Start starts the HTTP server and listens for incoming requests.
func (app *App) Start() error {
	return app.Server().ListenAndServe()
}

// NewApp initializes the application with the given Echo instance, version,
// and optional dependencies. Returns a pointer to the app instance
// with singleton behavior.
func NewApp(e *echo.Echo, config *internal.Config) *App {
	e.HideBanner = true

	api := e.Group("/api")

	app := &App{
		config:   config,
		echoApp:  e,
		apiGroup: api,
	}

	handler.UseCustomErrorHandler(app.Echo())
	handler.UseGlobalMiddlewares(app.Echo())

	return app
}
