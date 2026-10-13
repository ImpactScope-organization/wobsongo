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

// buildAppClaimCheckDeps bundles the claim-checking feature's dependencies
// for buildApp — chunkRepo/knowledgeRepo/embedder are already constructed in
// cmd/server.go for the River workers and are reused here as-is (RAGService
// only calls their read methods), rather than building second instances.
type buildAppClaimCheckDeps struct {
	chunkRepo     data.DocumentChunkRepoer
	knowledgeRepo data.AtomicKnowledgeRepoer
	embedder      data.Embedder
	claimAnalyzer data.ClaimAnalyzer
	claimJudge    data.ClaimJudge
}

// buildAppSpeechDeps bundles the speech (voice reply) feature's dependencies
// for buildApp — both are already constructed in cmd/server.go for the
// SpeechWorker and are reused here as-is. synthesizer is nil when
// MODAL_TTS_ENDPOINT is not configured.
type buildAppSpeechDeps struct {
	translator  data.Translator
	synthesizer data.SpeechSynthesizer
}

// buildApp initializes all API-facing repositories and returns a configured core.App.
// mediaProvider is constructed by the caller (cmd/server.go), shared with any
// River workers that also need it, rather than built again here.
func buildApp(
	config *internal.Config,
	pool *pgxpool.Pool,
	riverClient *river.Client[pgx.Tx],
	mediaProvider data.MediaUploadProvider,
	claimCheck buildAppClaimCheckDeps,
	speech buildAppSpeechDeps,
) *core.App {
	queries := db.New(pool)
	riverClientFn := func() *river.Client[pgx.Tx] { return riverClient }

	apifyRepo := repo.NewApifyRepo(riverClientFn)
	videoRepo := repo.NewVideoRepo(
		queries,
		pool,
		riverClientFn,
	)
	documentRepo := repo.NewDocumentRepo(db.New(pool), pool, riverClient)
	userRepo := repo.NewUserRepo(queries, pool)
	conversationRepo := repo.NewConversationRepo(queries, pool, riverClientFn)
	speechRepo := repo.NewSpeechRepo(riverClientFn)

	return core.NewApp(
		echo.New(),
		config,
		core.WithApifyRepo(apifyRepo),
		core.WithVideoRepo(videoRepo),
		core.WithDocumentRepo(documentRepo),
		core.WithMediaProvider(mediaProvider),
		core.WithChunkRepo(claimCheck.chunkRepo),
		core.WithKnowledgeRepo(claimCheck.knowledgeRepo),
		core.WithEmbedder(claimCheck.embedder),
		core.WithClaimAnalyzer(claimCheck.claimAnalyzer),
		core.WithClaimJudge(claimCheck.claimJudge),
		core.WithUserRepo(userRepo),
		core.WithConversationRepo(conversationRepo),
		core.WithSpeechRepo(speechRepo),
		core.WithTranslator(speech.translator),
		core.WithSpeechSynthesizer(speech.synthesizer),
	)
}
