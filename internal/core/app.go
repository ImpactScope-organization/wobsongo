package core

import (
	"fmt"
	"net/http"
	"time"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/handler"
	"github.com/impactscope-organization/wobsongo/internal/validation"
	"github.com/impactscope-organization/wobsongo/internal/webhandler"
	"github.com/impactscope-organization/wobsongo/ui"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	ReadTimeout  = 5 * time.Second
	WriteTimeout = 10 * time.Second
)

type App struct {
	config              *internal.Config
	echoApp             *echo.Echo
	apiGroup            *echo.Group
	apifyRepo           data.ApifyRepoer
	videoRepo           data.VideoRepoer
	documentRepo        data.DocumentRepoer
	mediaProvider       data.MediaUploadProvider
	chunkRepo           data.DocumentChunkRepoer
	knowledgeRepo       data.AtomicKnowledgeRepoer
	embedder            data.Embedder
	claimAnalyzer       data.ClaimAnalyzer
	claimJudge          data.ClaimJudge
	userRepo            data.UserRepoer
	conversationRepo    data.ConversationRepoer
	speechRepo          data.SpeechRepoer
	translator          data.Translator
	speechSynthesizer   data.SpeechSynthesizer
	narrativeRepo       data.NarrativeRepoer
	narrativeStatements data.NarrativeStatementGenerator
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

// WithUserRepo sets the user repository for the application's web layer.
func WithUserRepo(repo data.UserRepoer) AppOption {
	return func(a *App) {
		a.userRepo = repo
	}
}

func WithConversationRepo(repo data.ConversationRepoer) AppOption {
	return func(a *App) {
		a.conversationRepo = repo
	}
}

// WithSpeechRepo sets the speech-synthesis job repository for the application.
func WithSpeechRepo(repo data.SpeechRepoer) AppOption {
	return func(a *App) {
		a.speechRepo = repo
	}
}

// WithTranslator sets the translation client used to translate bot replies
// before they are spoken.
func WithTranslator(translator data.Translator) AppOption {
	return func(a *App) {
		a.translator = translator
	}
}

// WithSpeechSynthesizer sets the text-to-speech client for the application.
func WithSpeechSynthesizer(synthesizer data.SpeechSynthesizer) AppOption {
	return func(a *App) {
		a.speechSynthesizer = synthesizer
	}
}

// WithNarrativeRepo sets the repository of the claim-to-narrative flow.
func WithNarrativeRepo(repo data.NarrativeRepoer) AppOption {
	return func(a *App) {
		a.narrativeRepo = repo
	}
}

// WithNarrativeStatements sets the generator that writes the statement of a
// new narrative.
func WithNarrativeStatements(generator data.NarrativeStatementGenerator) AppOption {
	return func(a *App) {
		a.narrativeStatements = generator
	}
}

// General API info for swag (go tool swag init --generalInfo ./internal/core/app.go).
//
// @title						WobSongo API
// @version					1.0
// @description				Fact-checking backend for the WobSongo dashboard and messaging bots.
// @BasePath					/
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Access token from /api/v1/auth/login, as "Bearer <token>".
// @securityDefinitions.apikey	PSKAuth
// @in							header
// @name						Authorization
// @description				Pre-shared key for the messaging bots, as "PSK <key>".

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
	repos.ConversationRepo = app.conversationRepo
	repos.SpeechRepo = app.speechRepo
	repos.Translator = app.translator
	repos.SpeechSynthesizer = app.speechSynthesizer
	repos.UserRepo = app.userRepo
	repos.NarrativeRepo = app.narrativeRepo
	repos.NarrativeStatements = app.narrativeStatements

	handlers := handler.NewHandlers(repos)
	handlers.RegisterRoutes(app.apiGroup)

	// HTML routes, mounted alongside /api on the same Echo instance — token
	// comes from a cookie instead of the Authorization header.
	jwtAuth := auth.New(config.JWTSecret, config.JWTAccessTTL, config.JWTRefreshTTL)
	webGroup := e.Group("",
		handler.JWTFromCookieMiddleware(webhandler.AuthCookieName),
		handler.JWTParserMiddleware(jwtAuth),
		middleware.CSRFWithConfig(middleware.CSRFConfig{
			TokenLookup:    "form:_csrf",
			CookieSecure:   config.IsProduction(),
			CookieSameSite: http.SameSiteLaxMode,
		}),
	)
	webhandler.RegisterWebRoutes(
		webGroup, &webhandler.WebRepos{UserRepo: app.userRepo}, jwtAuth, config,
	)

	e.StaticFS("/static", ui.StaticFS)

	return app
}
