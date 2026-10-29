package handler

import (
	"fmt"
	"net/http"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/auth"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Handlers struct holds references to all the individual handler structs.
type Handlers struct {
	apifyHandler     *ApifyHandler
	documentHandler  *DocumentHandler
	mediaHandler     *MediaHandler
	botExtractPSK    string
	claimHandler     *ClaimHandler
	agentHandler     *AgentHandler
	ttsHandler       *TTSHandler
	authHandler      *AuthHandler
	jwtAuth          *auth.Auth
	config           *internal.Config
	narrativeHandler *NarrativeHandler
}

// RegisterRoutes registers all the API routes with their corresponding handlers.
func (h *Handlers) RegisterRoutes(api *echo.Group) {
	// apify Webhooks (legacy, unversioned)
	api.POST("/webhooks/apify", h.apifyHandler.webhookHandler)
	api.POST("/extract", h.apifyHandler.extractMediaHandler, PSKAuthMiddleware(h.botExtractPSK))
	api.POST(
		"/agent/inbound",
		h.agentHandler.inboundMessageHandler,
		PSKAuthMiddleware(h.botExtractPSK),
	)
	api.POST(
		"/agent/inbound-voice",
		h.agentHandler.inboundVoiceHandler,
		PSKAuthMiddleware(h.botExtractPSK),
	)
	api.POST(
		"/tts",
		h.ttsHandler.synthesizeHandler,
		PSKAuthMiddleware(h.botExtractPSK),
	)

	// Versioned resource routes use /api/v1/{resource_name_plural}.
	v1 := api.Group("/v1")

	// Dashboard users sign in here; everything else under /v1 that the
	// dashboard calls needs the access token as a bearer token.
	limited := authRateLimit(h.config)
	v1.POST("/auth/login", h.authHandler.loginHandler, limited)
	v1.POST("/auth/signup", h.authHandler.signupHandler, limited)
	v1.POST("/auth/refresh", h.authHandler.refreshHandler, limited)

	signedIn := BearerAuthMiddleware(h.jwtAuth)
	v1.GET("/me", h.authHandler.meHandler, signedIn)
	v1.PATCH("/me", h.authHandler.updateProfileHandler, signedIn)
	v1.POST("/me/password", h.authHandler.changePasswordHandler, signedIn)

	admin := RequireRoleMiddleware(model.RoleAdmin)
	v1.POST("/documents", h.documentHandler.createDocumentHandler, signedIn, admin)
	v1.GET("/documents", h.documentHandler.listDocumentsHandler, signedIn, admin)
	v1.GET("/documents/:id", h.documentHandler.getDocumentHandler, signedIn, admin)
	v1.PUT("/documents/:id", h.documentHandler.updateDocumentHandler, signedIn, admin)
	v1.DELETE("/documents/:id", h.documentHandler.deleteDocumentHandler, signedIn, admin)

	v1.GET(
		"/media/upload",
		h.mediaHandler.getPresignedPOSTURL,
		PSKAuthMiddleware(h.botExtractPSK),
	)
	v1.GET(
		"/media/presigned-url",
		h.mediaHandler.getPresignedGETURL,
		PSKAuthMiddleware(h.botExtractPSK),
	)

	v1.POST("/claims/check", h.claimHandler.checkClaimHandler, signedIn)

	v1.GET("/narratives", h.narrativeHandler.listNarrativesHandler, signedIn)
	v1.GET("/narratives/:id", h.narrativeHandler.getNarrativeHandler, signedIn)
}

// Repos holds the repository interfaces required by the handlers.
type Repos struct {
	ApifyRepo           data.ApifyRepoer
	VideoRepo           data.VideoRepoer
	DocumentRepo        data.DocumentRepoer
	MediaProvider       data.MediaUploadProvider
	ChunkRepo           data.DocumentChunkRepoer
	KnowledgeRepo       data.AtomicKnowledgeRepoer
	Embedder            data.Embedder
	ClaimAnalyzer       data.ClaimAnalyzer
	ClaimJudge          data.ClaimJudge
	ConversationRepo    data.ConversationRepoer
	SpeechRepo          data.SpeechRepoer
	Translator          data.Translator
	SpeechSynthesizer   data.SpeechSynthesizer
	UserRepo            data.UserRepoer
	NarrativeRepo       data.NarrativeRepoer
	NarrativeStatements data.NarrativeStatementGenerator
}

// NewHandlers creates a new Handlers instance with the provided repositories.
// Dependency injection is used to provide the necessary services to the handlers.
// You can provide mock implementations of the repositories for testing purposes.
func NewHandlers(repos *Repos) *Handlers {
	config := internal.NewConfig()
	// Initialize Apify services and handlers
	videoService := service.NewVideoService(repos.VideoRepo)

	scheme := config.APISchemes()[0]
	apifyService := service.NewApifyService(
		repos.ApifyRepo,
		repos.VideoRepo,
		videoService,
		http.DefaultClient,
		config.ApifyConfig.Token,
		scheme+"://"+config.APIHost,
	)
	apifyHandler := NewApifyHandler(apifyService)

	// Initialize Document services and handlers
	documentService := service.NewDocumentService(repos.DocumentRepo)
	documentHandler := NewDocumentHandler(documentService)

	// Initialize Media services and handlers
	mediaService := service.NewMediaService(repos.MediaProvider)
	mediaHandler := NewMediaHandler(mediaService)

	// Initialize claim-checking services and handlers
	ragService := service.NewRAGService(repos.ChunkRepo, repos.KnowledgeRepo, repos.Embedder)
	claimService := service.NewClaimService(repos.ClaimAnalyzer, repos.ClaimJudge, ragService)

	var narrativeService *service.NarrativeService
	if repos.NarrativeRepo != nil {
		narrativeService = service.NewNarrativeService(
			repos.NarrativeRepo,
			repos.Embedder,
			repos.NarrativeStatements,
			config.NarrativeConfig.MatchThreshold,
		)
	}
	if narrativeService != nil && repos.NarrativeStatements != nil {
		claimService = claimService.WithNarratives(narrativeService)
	}
	claimHandler := NewClaimHandler(claimService)

	var narrativeHandler *NarrativeHandler
	if narrativeService != nil {
		narrativeHandler = NewNarrativeHandler(narrativeService)
	}

	agentLLMConfig, err := internal.NewAgentLLMConfig()
	if err != nil {
		panic(fmt.Errorf("failed to load agent LLM config: %w", err))
	}
	botClient := external.NewBotClient(
		config.BotBaseURL,
		config.BotCallbackPSK,
		config.BotExtractPSK,
	)
	conversationService := service.NewConversationService(repos.ConversationRepo)
	agentService, err := service.NewAgentService(
		conversationService,
		apifyService,
		claimService,
		nil,
		botClient,
		agentLLMConfig.Enabled,
		agentLLMConfig.Provider,
		agentLLMConfig.Model,
		agentLLMConfig.BaseURL,
		agentLLMConfig.APIKey,
	)
	if err != nil {
		panic(fmt.Errorf("failed to build agent service: %w", err))
	}
	agentHandler := NewAgentHandler(agentService)

	// Initialize speech (TTS) service and handler.
	speechService := service.NewSpeechService(
		repos.Translator,
		repos.SpeechSynthesizer,
		repos.SpeechRepo,
	)
	ttsHandler := NewTTSHandler(speechService)

	jwtAuth := auth.New(config.JWTSecret, config.JWTAccessTTL, config.JWTRefreshTTL)
	authHandler := NewAuthHandler(service.NewAuthService(repos.UserRepo, jwtAuth))

	return &Handlers{
		apifyHandler:     apifyHandler,
		documentHandler:  documentHandler,
		mediaHandler:     mediaHandler,
		ttsHandler:       ttsHandler,
		botExtractPSK:    config.BotExtractPSK,
		claimHandler:     claimHandler,
		agentHandler:     agentHandler,
		authHandler:      authHandler,
		jwtAuth:          jwtAuth,
		config:           config,
		narrativeHandler: narrativeHandler,
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
