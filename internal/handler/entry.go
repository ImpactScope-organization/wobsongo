package handler

import (
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Handlers struct holds references to all the individual handler structs.
type Handlers struct {
	apifyHandler *ApifyHandler
}

// RegisterRoutes registers all the API routes with their corresponding handlers.
func (h *Handlers) RegisterRoutes(api *echo.Group) {
	// apify Webhooks
	api.POST("/webhooks/apify", h.apifyHandler.webhookHandler)
	api.POST("/extract", h.apifyHandler.extractMediaHandler)
}

// Repos holds the repository interfaces required by the handlers.
type Repos struct {
	ApifyRepo data.ApifyRepoer
	VideoRepo data.VideoRepoer
}

// NewHandlers creates a new Handlers instance with the provided repositories.
// Dependency injection is used to provide the necessary services to the handlers.
// You can provide mock implementations of the repositories for testing purposes.
func NewHandlers(repos *Repos) *Handlers {
	config := internal.NewConfig()
	// Initialize Apify services and handlers
	videoService := service.NewVideoService(repos.VideoRepo)
	apifyService := service.NewApifyService(
		repos.ApifyRepo,
		videoService,
		http.DefaultClient,
		config.ApifyToken,
	)
	apifyHandler := NewApifyHandler(apifyService)

	return &Handlers{
		apifyHandler: apifyHandler,
	}
}

// UseGlobalMiddlewares attaches default global middleware handlers to the given Echo instance.
func UseGlobalMiddlewares(e *echo.Echo) {
	/*
		The order of middleware is important.
		Recover should be the first to catch panics.
		requestIDMiddleware should be early to assign request IDs.
		corsMiddleware should be before any handlers that need CORS.
		loggerMiddleware should log all requests.
	*/
	e.Use(
		middleware.Recover(),
		requestIDMiddleware,
		corsMiddleware(),
		loggerMiddleware(),
	)
}

// UseCustomErrorHandler sets a custom error handler for the given Echo instance.
func UseCustomErrorHandler(e *echo.Echo) {
	e.HTTPErrorHandler = customErrorHandler()
}
