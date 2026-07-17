package cmd

import (
	"os"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/spf13/cobra"
)

var claimCheckCmd = &cobra.Command{
	Use:   "claim-check [claim]",
	Short: "Check a claim against the knowledge base and print the judged verdict",
	Long: "Runs the full claim-checking pipeline — scope/decomposition, per-sub-claim\n" +
		"hybrid-search retrieval (reusing the same retrieval layer as `wob rag`), and\n" +
		"verdict judging — against the real dev DB and configured LLM endpoints.\n" +
		"Read-only: no --apply flag, nothing is mutated.",
	Args: cobra.ExactArgs(1),
	Run:  runClaimCheck,
}

func runClaimCheck(cmd *cobra.Command, args []string) {
	claim := args[0]
	config := internal.NewConfig(EnvFile)

	if err := internal.IsEmbeddingOK(config.EmbeddingConfig); err != nil {
		cmd.PrintErrf("Config error: %s\n", err.Error())
		os.Exit(1)
		return
	}
	if err := internal.IsClaimCheckOK(config.ClaimCheckConfig); err != nil {
		cmd.PrintErrf("Config error: %s\n", err.Error())
		os.Exit(1)
		return
	}

	ctx := cmd.Context()
	pool, err := repo.NewPgxPool(ctx, config.PostgresURI)
	if err != nil {
		cmd.PrintErrf("Failed to connect to database: %s\n", err.Error())
		os.Exit(1)
		return
	}
	defer pool.Close()

	// nil riverClient: this command only reads, never enqueues — same
	// justification as cmd/rag.go's chunkRepo.
	chunkRepo := repo.NewDocumentChunkRepo(db.New(pool), pool, nil)
	knowledgeRepo := repo.NewAtomicKnowledgeRepo(db.New(pool), pool)
	embeddingClient := newEmbeddingClient(config.EmbeddingConfig)

	analyzerClient := external.NewClaimAnalyzerClient(
		config.ClaimCheckConfig.BaseURL,
		config.ClaimCheckConfig.Model,
		config.ClaimCheckConfig.APIKey,
	)
	judgeClient := external.NewJudgeClient(
		config.ClaimCheckConfig.BaseURL,
		config.ClaimCheckConfig.Model,
		config.ClaimCheckConfig.APIKey,
	)

	ragService := service.NewRAGService(chunkRepo, knowledgeRepo, embeddingClient)
	claimService := service.NewClaimService(analyzerClient, judgeClient, ragService)

	result, err := claimService.CheckClaim(ctx, &dto.CheckClaimDTO{Text: claim})
	if err != nil {
		cmd.PrintErrf("Claim check failed: %s\n", err.Error())
		os.Exit(1)
		return
	}

	printClaimCheckResult(cmd, result)
}

func printClaimCheckResult(cmd *cobra.Command, result *service.ClaimCheckResult) {
	if !result.InScope {
		cmd.Printf("Out of scope: %s\n", result.RefusalReason)
		return
	}

	cmd.Printf("Overall: %s\n\n", result.OverallSummary)
	for i, sc := range result.SubClaims {
		cmd.Printf("%d. %q\n", i+1, sc.Claim)
		cmd.Printf(
			"   verdict=%s severity=%s recommend_medical_consult=%t\n",
			sc.Verdict, sc.Severity, sc.RecommendMedicalConsult,
		)
		if sc.Reasoning != "" {
			cmd.Printf("   reasoning: %s\n", sc.Reasoning)
		}
		if len(sc.Citations) == 0 {
			cmd.Printf("   citations: (none)\n")
		} else {
			cmd.Printf("   citations: (matches the [N] references in reasoning above)\n")
			for _, c := range sc.Citations {
				cmd.Printf(
					"     [%d] (%s) doc=%s lang=%s %s\n",
					c.Index, c.Source, c.DocumentID, c.Language, truncateForDisplay(c.Text, 200),
				)
			}
		}
		cmd.Println()
	}
}
