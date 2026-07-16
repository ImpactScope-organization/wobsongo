package core

import (
	"fmt"
	"net/http"
	"time"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/handler"
	"github.com/impactscope-organization/wobsongo/internal/validation"
	"github.com/labstack/echo/v4"
)

const (
	ReadTimeout  = 5 * time.Second
	WriteTimeout = 10 * time.Second
)

type App struct {
	config        *internal.Config
	echoApp       *echo.Echo
	apiGroup      *echo.Group
	apifyRepo     data.ApifyRepoer
	videoRepo     data.VideoRepoer
	documentRepo  data.DocumentRepoer
	mediaProvider data.MediaUploadProvider
	chunkRepo     data.DocumentChunkRepoer
	knowledgeRepo data.AtomicKnowledgeRepoer
	embedder      data.Embedder
	claimAnalyzer data.ClaimAnalyzer
	claimJudge    data.ClaimJudge
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

// AppOption defines a function type for configuring the App with optional dependencies.
type AppOption func(*App)

// WithApifyRepo sets the Apify repository for the application.
func WithApifyRepo(repo data.ApifyRepoer) AppOption {
	return func(a *App) {
		a.apifyRepo = repo
	}
}

// WithVideoRepo sets the Video repository for the application.
func WithVideoRepo(repo data.VideoRepoer) AppOption {
	return func(a *App) {
		a.videoRepo = repo
	}
}

// WithDocumentRepo sets the document repository for the application.
func WithDocumentRepo(repo data.DocumentRepoer) AppOption {
	return func(a *App) {
		a.documentRepo = repo
	}
}

// WithMediaProvider sets the media upload provider for the application.
func WithMediaProvider(provider data.MediaUploadProvider) AppOption {
	return func(a *App) {
		a.mediaProvider = provider
	}
}

// WithChunkRepo sets the document chunk repository for the application.
func WithChunkRepo(repo data.DocumentChunkRepoer) AppOption {
	return func(a *App) {
		a.chunkRepo = repo
	}
}

// WithKnowledgeRepo sets the atomic knowledge repository for the application.
func WithKnowledgeRepo(repo data.AtomicKnowledgeRepoer) AppOption {
	return func(a *App) {
		a.knowledgeRepo = repo
	}
}

// WithEmbedder sets the embedding client used for hybrid-search retrieval.
func WithEmbedder(embedder data.Embedder) AppOption {
	return func(a *App) {
		a.embedder = embedder
	}
}

// WithClaimAnalyzer sets the claim scope/decomposition analyzer for the application.
func WithClaimAnalyzer(analyzer data.ClaimAnalyzer) AppOption {
	return func(a *App) {
		a.claimAnalyzer = analyzer
	}
}

// WithClaimJudge sets the claim judge for the application.
func WithClaimJudge(judge data.ClaimJudge) AppOption {
	return func(a *App) {
		a.claimJudge = judge
	}
}

// NewApp initializes the application with the given Echo instance, version,
// and optional dependencies. Returns a pointer to the app instance
// with singleton behavior.
func NewApp(e *echo.Echo, config *internal.Config, optionFuncs ...AppOption) *App {
	e.HideBanner = true

	api := e.Group("/api")

	app := &App{
		config:   config,
		echoApp:  e,
		apiGroup: api,
	}

	handler.UseCustomErrorHandler(app.Echo())
	handler.UseGlobalMiddlewares(app.Echo())

	if err := validation.Register(app.Echo()); err != nil {
		panic(fmt.Errorf("failed to register DTO validator: %w", err))
	}

	// Run option functions to set optional dependencies.
	for _, optionFunc := range optionFuncs {
		optionFunc(app)
	}

	// Initialize repositories and handlers.
	repos := new(handler.Repos)
	repos.ApifyRepo = app.apifyRepo
	repos.VideoRepo = app.videoRepo
	repos.DocumentRepo = app.documentRepo
	repos.MediaProvider = app.mediaProvider
	repos.ChunkRepo = app.chunkRepo
	repos.KnowledgeRepo = app.knowledgeRepo
	repos.Embedder = app.embedder
	repos.ClaimAnalyzer = app.claimAnalyzer
	repos.ClaimJudge = app.claimJudge

	handlers := handler.NewHandlers(repos)
	handlers.RegisterRoutes(app.apiGroup)

	return app
}
