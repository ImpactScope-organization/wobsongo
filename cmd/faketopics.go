package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/spf13/cobra"
)

var fakeTopicsCmd = &cobra.Command{
	Use:   "faketopics",
	Short: "Seed the closed topic list for non-production environments",
	Long: `Seed a fixed set of health topics for development and testing.

This command is ONLY available in non-production environments.
Topics are upserted by name, so re-running is safe: an existing topic keeps its
ID and gets its description updated and is set back to active.

By default, this runs in DRY-RUN mode and shows what would be created.
Use --apply flag to actually write to the database.

Example:
  # Preview the topics (dry-run)
  go run . -e .env.dev faketopics

  # Create the topics
  go run . -e .env.dev faketopics --apply`,

	Run: runFakeTopics,
}

func init() {
	fakeTopicsCmd.Flags().
		Bool("apply", false, "Actually write the topics (default: dry-run)")
}

// fakeTopicSeed is one topic to seed. The names are also the keys of
// fakeNarrativeStatements in fakenarratives.go.
type fakeTopicSeed struct {
	name        string
	description string
}

var fakeTopicSeeds = []fakeTopicSeed{
	{
		name:        "Contraception and Family Planning",
		description: "Methods, effectiveness, side effects and myths about contraception and birth spacing.",
	},
	{
		name:        "HIV/AIDS and STIs",
		description: "Transmission, testing, prevention and treatment of HIV and sexually transmitted infections.",
	},
	{
		name:        "Pregnancy and Antenatal Care",
		description: "Pregnancy health, antenatal check-ups, safe supplements and childbirth preparation.",
	},
	{
		name:        "Safe Abortion and Post-Abortion Care",
		description: "Safe abortion, unsafe methods, complications and care after abortion or miscarriage.",
	},
	{
		name:        "Menstrual Health",
		description: "The menstrual cycle, period pain, irregular periods, hygiene and menstrual myths.",
	},
	{
		name:        "Adolescent Sexual Health",
		description: "Puberty, first sexual experiences, adolescent pregnancy and sexual health education for young people.",
	},
	{
		name:        "Fertility and Infertility",
		description: "Conception, causes of infertility in women and men, and fertility treatments and remedies.",
	},
	{
		name:        "HPV and Reproductive Cancers",
		description: "HPV vaccination, cervical cancer screening and other reproductive cancers.",
	},
}

func runFakeTopics(cmd *cobra.Command, _ []string) {
	apply, _ := cmd.Flags().GetBool("apply")

	config := internal.NewConfig(EnvFile)

	// Safety check: prevent running in production.
	if config.IsProduction() {
		printFakeTopicsProductionError(cmd)
		os.Exit(1)
		return
	}

	printFakeTopicsHeader(cmd, apply)

	if !apply {
		printFakeTopicsPreview(cmd)
		return
	}

	if err := applyFakeTopics(cmd, config.PostgresURI); err != nil {
		cmd.PrintErrf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func printFakeTopicsProductionError(cmd *cobra.Command) {
	cmd.PrintErrln("╔════════════════════════════════════════════════════════════════╗")
	cmd.PrintErrln("║  ERROR: Cannot Generate Fake Data in Production                ║")
	cmd.PrintErrln("╚════════════════════════════════════════════════════════════════╝")
	cmd.PrintErrln()
	cmd.PrintErrln("This command is only available in development/testing environments.")
}

func printFakeTopicsHeader(cmd *cobra.Command, apply bool) {
	cmd.Println("╔════════════════════════════════════════════════════════════════╗")
	if apply {
		cmd.Println("║  Creating Fake Topics                                          ║")
	} else {
		cmd.Println("║  DRY RUN: Preview Fake Topics                                  ║")
	}
	cmd.Println("╚════════════════════════════════════════════════════════════════╝")
	cmd.Println()
	cmd.Printf("Topics to upsert: %d\n", len(fakeTopicSeeds))
	cmd.Println()
}

func printFakeTopicsPreview(cmd *cobra.Command) {
	cmd.Println("Preview of topics:")
	cmd.Println()
	for i, seed := range fakeTopicSeeds {
		cmd.Printf("  %d. %s\n", i+1, seed.name)
		cmd.Printf("     %s\n", seed.description)
		cmd.Println()
	}
	cmd.Println("✓ This is a DRY RUN - no changes were made")
	cmd.Println("✓ To create these topics, add the --apply flag")
	cmd.Println()
	cmd.Println("Example:")
	if EnvFile != "" {
		cmd.Printf("  go run . -e %s faketopics --apply\n", EnvFile)
	} else {
		cmd.Println("  go run . faketopics --apply")
	}
}

// applyFakeTopics upserts all topics in one transaction (atomic).
func applyFakeTopics(cmd *cobra.Command, postgresURI string) error {
	ctx := cmd.Context()

	pool, err := repo.NewPgxPool(ctx, postgresURI)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	cmd.Println("Creating topics...")
	cmd.Println()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		// Rollback on any error (no-op if already committed).
		_ = tx.Rollback(ctx)
	}()

	txQueries := db.New(pool).WithTx(tx)

	for i, seed := range fakeTopicSeeds {
		topic, err := txQueries.UpsertTopicByName(ctx, db.UpsertTopicByNameParams{
			Name:        seed.name,
			Description: seed.description,
			IsActive:    true,
		})
		if err != nil {
			cmd.Println()
			cmd.PrintErrf("❌ Error upserting topic '%s': %v\n", seed.name, err)
			cmd.Println()
			cmd.Println("⚠️  Transaction rolled back - no topics were written.")
			return errors.New("failed to upsert topics")
		}

		cmd.Printf(
			"  ✅ [%d/%d] Upserted: %s (ID: %s)\n",
			i+1,
			len(fakeTopicSeeds),
			topic.Name,
			topic.ID,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		cmd.Println()
		cmd.Println("⚠️  Transaction rolled back - no topics were written.")
		return fmt.Errorf("error committing transaction: %w", err)
	}

	printFakeTopicsSummary(cmd)
	return nil
}

func printFakeTopicsSummary(cmd *cobra.Command) {
	cmd.Println()
	cmd.Println("╔════════════════════════════════════════════════════════════════╗")
	cmd.Println("║  Summary                                                       ║")
	cmd.Println("╚════════════════════════════════════════════════════════════════╝")
	cmd.Println()
	cmd.Printf("✅ Successfully upserted: %d topics\n", len(fakeTopicSeeds))
	cmd.Println()
	cmd.Println("All topics have been committed to the database.")
}
