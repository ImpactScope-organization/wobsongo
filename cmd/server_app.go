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

// buildApp initializes all API-facing repositories and returns a configured core.App.
// mediaProvider is constructed by the caller (cmd/server.go), shared with any
// River workers that also need it, rather than built again here.
func buildApp(
	config *internal.Config,
	pool *pgxpool.Pool,
	riverClient *river.Client[pgx.Tx],
	mediaProvider data.MediaUploadProvider,
	claimCheck buildAppClaimCheckDeps,
) *core.App {
	queries := db.New(pool)

	apifyRepo := repo.NewApifyRepo(riverClient)
	videoRepo := repo.NewVideoRepo(
		queries,
		pool,
		riverClient,
	)
	documentRepo := repo.NewDocumentRepo(db.New(pool), pool, riverClient)
	userRepo := repo.NewUserRepo(queries, pool)

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
	)
}
