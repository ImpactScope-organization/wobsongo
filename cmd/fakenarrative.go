package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

var fakeNarrativesCmd = &cobra.Command{
	Use:   "fakenarratives",
	Short: "Generate fake narratives for non-production environments",
	Long: `Generate fake narratives under the existing active topics for development and testing.

This command is ONLY available in non-production environments.
Topics must exist first: run the faketopics command beforehand.
Narratives whose statement already exists under the same topic are skipped, so
re-running does not create duplicates.

By default, this runs in DRY-RUN mode and shows what would be created.
Use --apply flag to actually write to the database.

Example:
  # Preview 2 narratives per topic (dry-run)
  go run . -e .env.dev fakenarratives

  # Create 2 narratives per topic
  go run . -e .env.dev fakenarratives --apply`,

	Run: runFakeNarratives,
}

func init() {
	fakeNarrativesCmd.Flags().IntP("count", "n", 2, "Number of narratives to generate per topic")
	fakeNarrativesCmd.Flags().
		Bool("apply", false, "Actually write the narratives (default: dry-run)")
}

// fakeNarrativeStatements holds realistic, deliberately broad narrative
// statements keyed by the topic names seeded in faketopics.go.
var fakeNarrativeStatements = map[string][]string{
	"Contraception and Family Planning": {
		"Contraceptive pills or implants cause permanent infertility.",
		"Withdrawal or washing after sex reliably prevents pregnancy.",
	},
	"HIV/AIDS and STIs": {
		"HIV is spread by mosquito bites or by sharing meals.",
		"An infection that shows no symptoms is not a real infection, so testing is not needed.",
	},
	"Pregnancy and Antenatal Care": {
		"Herbal supplements during pregnancy are always safe for the baby.",
		"Antenatal check-ups are only needed if something feels wrong.",
	},
	"Safe Abortion and Post-Abortion Care": {
		"Home-made herbal or drug mixtures are a safe way to end a pregnancy.",
		"A woman who has had an abortion can never have children again.",
	},
	"Menstrual Health": {
		"Menstruating women are unclean and must not prepare food or touch certain objects.",
		"Severe period pain or very irregular periods are normal and never need medical attention.",
	},
	"Adolescent Sexual Health": {
		"A girl cannot get pregnant the first time she has sex.",
		"Masturbation causes infertility or serious illness.",
	},
	"Fertility and Infertility": {
		"Infertility is always the woman's fault.",
		"Herbal mixtures or vaginal steaming can cure infertility.",
	},
	"HPV and Reproductive Cancers": {
		"The HPV vaccine encourages girls to start having sex.",
		"Cervical cancer screening is unnecessary if you have no symptoms.",
	},
}

// fakeNarrativeStatuses are the statuses picked at random for fake narratives.
var fakeNarrativeStatuses = []model.NarrativeStatus{
	model.NarrativeStatusNeedsCheck,
	model.NarrativeStatusAutoChecked,
	model.NarrativeStatusNotInKB,
	model.NarrativeStatusExpertVerified,
}

// fakeNarrativePlan is one narrative about to be created.
type fakeNarrativePlan struct {
	statement     string
	harm          pgtype.Int2
	harmValidated bool
	status        model.NarrativeStatus
	isActive      bool
}

func runFakeNarratives(cmd *cobra.Command, _ []string) {
	count, _ := cmd.Flags().GetInt("count")
	apply, _ := cmd.Flags().GetBool("apply")

	if count <= 0 {
		cmd.PrintErrln("Error: --count must be greater than 0")
		os.Exit(1)
		return
	}

	config := internal.NewConfig(EnvFile)

	// Safety check: prevent running in production.
	if config.IsProduction() {
		printFakeNarrativesProductionError(cmd)
		os.Exit(1)
		return
	}

	if err := generateFakeNarratives(cmd, config.PostgresURI, count, apply); err != nil {
		cmd.PrintErrf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func printFakeNarrativesProductionError(cmd *cobra.Command) {
	cmd.PrintErrln("╔════════════════════════════════════════════════════════════════╗")
	cmd.PrintErrln("║  ERROR: Cannot Generate Fake Data in Production                ║")
	cmd.PrintErrln("╚════════════════════════════════════════════════════════════════╝")
	cmd.PrintErrln()
	cmd.PrintErrln("This command is only available in development/testing environments.")
}

// generateFakeNarratives loads the topics, builds the plan, then either
// previews it (dry-run) or writes it to the database.
func generateFakeNarratives(cmd *cobra.Command, postgresURI string, count int, apply bool) error {
	ctx := cmd.Context()

	pool, err := repo.NewPgxPool(ctx, postgresURI)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	topics, err := queries.ListActiveTopics(ctx)
	if err != nil {
		return fmt.Errorf("failed to load topics: %w", err)
	}
	if len(topics) == 0 {
		return errors.New("no active topics found. Run the faketopics command first")
	}

	plans := buildFakeNarrativePlans(topics, count)

	printFakeNarrativesHeader(cmd, apply, len(topics), count)

	if !apply {
		printFakeNarrativesPreview(cmd, topics, plans, count)
		return nil
	}

	return applyFakeNarratives(cmd, pool, queries, topics, plans)
}

// buildFakeNarrativePlans builds one list of narratives per topic.
func buildFakeNarrativePlans(topics []db.Topic, count int) [][]fakeNarrativePlan {
	faker := gofakeit.New(uint64(time.Now().UnixNano()))

	plans := make([][]fakeNarrativePlan, len(topics))
	for t, topic := range topics {
		predefined := fakeNarrativeStatements[topic.Name]
		plans[t] = make([]fakeNarrativePlan, count)
		for i := range count {
			plans[t][i] = newFakeNarrativePlan(faker, predefined, i)
		}
	}
	return plans
}

func newFakeNarrativePlan(
	faker *gofakeit.Faker,
	predefined []string,
	i int,
) fakeNarrativePlan {
	statement := faker.Sentence(12)
	if i < len(predefined) {
		statement = predefined[i]
	}

	status := fakeNarrativeStatuses[faker.Number(0, len(fakeNarrativeStatuses)-1)]

	// A narrative without an estimate has a nil harm (about 1 in 5).
	var harm pgtype.Int2
	if faker.Number(1, 5) != 1 || status == model.NarrativeStatusExpertVerified {
		harm = pgtype.Int2{
			Int16: int16(faker.Number(model.NarrativeHarmMin, model.NarrativeHarmMax)),
			Valid: true,
		}
	}

	return fakeNarrativePlan{
		statement: statement,
		harm:      harm,
		// Only an expert-verified narrative has a validated harm.
		harmValidated: status == model.NarrativeStatusExpertVerified && harm.Valid,
		status:        status,
		isActive:      faker.Number(1, 5) != 1,
	}
}

func printFakeNarrativesHeader(cmd *cobra.Command, apply bool, topicCount, count int) {
	cmd.Println("╔════════════════════════════════════════════════════════════════╗")
	if apply {
		cmd.Println("║  Creating Fake Narratives                                      ║")
	} else {
		cmd.Println("║  DRY RUN: Preview Fake Narratives                              ║")
	}
	cmd.Println("╚════════════════════════════════════════════════════════════════╝")
	cmd.Println()
	cmd.Printf("Active topics: %d | Narratives per topic: %d\n", topicCount, count)
	cmd.Println()
}

func printFakeNarrativesPreview(
	cmd *cobra.Command,
	topics []db.Topic,
	plans [][]fakeNarrativePlan,
	count int,
) {
	cmd.Println("Preview of narratives:")
	cmd.Println()
	for t, topic := range topics {
		cmd.Printf("  %d. %s\n", t+1, topic.Name)
		for _, p := range plans[t] {
			cmd.Printf(
				"     - [%s] harm=%s active=%t\n",
				p.status,
				harmLabel(p.harm),
				p.isActive,
			)
			cmd.Printf("       %s\n", p.statement)
		}
		cmd.Println()
	}
	cmd.Println("✓ This is a DRY RUN - no changes were made")
	cmd.Println("✓ To create these narratives, add the --apply flag")
	cmd.Println()
	cmd.Println("Example:")
	if EnvFile != "" {
		cmd.Printf("  go run . -e %s fakenarratives --count=%d --apply\n", EnvFile, count)
	} else {
		cmd.Printf("  go run . fakenarratives --count=%d --apply\n", count)
	}
}

// applyFakeNarratives writes all narratives in one transaction (atomic).
func applyFakeNarratives(
	cmd *cobra.Command,
	pool *pgxpool.Pool,
	queries *db.Queries,
	topics []db.Topic,
	plans [][]fakeNarrativePlan,
) error {
	ctx := cmd.Context()

	cmd.Println("Creating narratives...")
	cmd.Println()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		// Rollback on any error (no-op if already committed).
		_ = tx.Rollback(ctx)
	}()

	txQueries := queries.WithTx(tx)

	created, skipped := 0, 0
	for t := range topics {
		topic := &topics[t]
		cmd.Printf("  Topic: %s\n", topic.Name)

		for _, p := range plans[t] {
			wasCreated, err := insertFakeNarrativeIfNew(ctx, cmd, txQueries, topic, p)
			if err != nil {
				cmd.PrintErrf("❌ %s\n", err.Error())
				cmd.Println("⚠️  Transaction rolled back - no narratives were written.")
				return errors.New("failed to create narratives")
			}
			if wasCreated {
				created++
			} else {
				skipped++
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		cmd.Println()
		cmd.Println("⚠️  Transaction rolled back - no narratives were written.")
		return fmt.Errorf("error committing transaction: %w", err)
	}

	printFakeNarrativesSummary(cmd, created, skipped)
	return nil
}

// insertFakeNarrativeIfNew inserts the narrative unless its statement already
// exists under the topic. It reports whether a row was created.
func insertFakeNarrativeIfNew(
	ctx context.Context,
	cmd *cobra.Command,
	q *db.Queries,
	topic *db.Topic,
	p fakeNarrativePlan,
) (bool, error) {
	exists, err := q.NarrativeExistsForTopic(ctx, db.NarrativeExistsForTopicParams{
		TopicID:   topic.ID,
		Statement: p.statement,
	})
	if err != nil {
		return false, fmt.Errorf("error checking narrative '%s': %w", p.statement, err)
	}
	if exists {
		cmd.Printf("     ↷ Skipped (already exists): %s\n", p.statement)
		return false, nil
	}

	narrative, err := q.InsertNarrative(ctx, db.InsertNarrativeParams{
		TopicID:       topic.ID,
		Statement:     p.statement,
		Harm:          p.harm,
		HarmValidated: p.harmValidated,
		Status:        string(p.status),
		IsActive:      p.isActive,
	})
	if err != nil {
		return false, fmt.Errorf("error creating narrative '%s': %w", p.statement, err)
	}

	cmd.Printf("     ✅ Created (ID: %s) [%s]: %s\n", narrative.ID, p.status, p.statement)
	return true, nil
}

func printFakeNarrativesSummary(cmd *cobra.Command, created, skipped int) {
	cmd.Println()
	cmd.Println("╔════════════════════════════════════════════════════════════════╗")
	cmd.Println("║  Summary                                                       ║")
	cmd.Println("╚════════════════════════════════════════════════════════════════╝")
	cmd.Println()
	cmd.Printf(
		"✅ Successfully created: %d narratives (skipped as duplicates: %d)\n",
		created,
		skipped,
	)
	cmd.Println()
	cmd.Println("All narratives have been committed to the database.")
}

// harmLabel renders a nullable harm level for the dry-run preview.
func harmLabel(harm pgtype.Int2) string {
	if !harm.Valid {
		return "-"
	}
	return strconv.Itoa(int(harm.Int16))
}
