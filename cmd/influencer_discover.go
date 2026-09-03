package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/spf13/cobra"
)

var influencerDiscoverKeywords string

var influencerDiscoverCmd = &cobra.Command{
	Use:   "influencer-discover",
	Short: "Search TikTok for new influencers talking about the configured topic keywords",
	Run:   runInfluencerDiscover,
}

func init() {
	influencerDiscoverCmd.Flags().StringVar(
		&influencerDiscoverKeywords,
		"keywords",
		"",
		"comma-separated search keywords (default: INFLUENCER_DISCOVERY_KEYWORDS / dummy default list)",
	)
}

func runInfluencerDiscover(cmd *cobra.Command, _ []string) {
	config := internal.NewConfig(EnvFile)

	if err := internal.IsInfluencerMonitorOK(config.ApifyConfig); err != nil {
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

	queries := db.New(pool)
	influencerRepo := repo.NewInfluencerRepo(queries, pool)
	influencerVideoRepo := repo.NewInfluencerVideoRepo(queries, pool)
	scraperClient := external.NewTikTokScraperClient(
		config.ApifyConfig.Token,
		config.ApifyConfig.TikTokActorID,
	)

	monitorService := service.NewInfluencerService(
		influencerRepo,
		influencerVideoRepo,
		scraperClient,
	)

	keywords := config.InfluencerMonitorConfig.DiscoveryKeywords
	if influencerDiscoverKeywords != "" {
		keywords = splitAndTrim(influencerDiscoverKeywords)
	}

	cmd.Printf("Searching %d keyword(s)...\n\n", len(keywords))

	results, err := monitorService.DiscoverInfluencers(
		ctx,
		keywords,
		config.InfluencerMonitorConfig.ResultsPerKeyword,
	)
	if err != nil {
		cmd.PrintErrf("Discovery run failed: %s\n", err.Error())
		os.Exit(1)
		return
	}

	printDiscoveryResults(cmd, results)
}

func printDiscoveryResults(cmd *cobra.Command, results []service.InfluencerDiscoveryResult) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "KEYWORD\tUSERNAME\tSTATUS\tSOURCE VIDEO")

	newCount := 0
	for _, r := range results {
		username := "@" + r.Username
		sourceVideo := r.SourceVideoURL
		status := "new"

		if r.Error != "" {
			username = "-"
			sourceVideo = "-"
			status = "error: " + r.Error
		} else if r.AlreadyKnown {
			status = "already tracked"
		} else {
			newCount++
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Keyword, username, status, sourceVideo)
	}
	_ = w.Flush()

	cmd.Printf(
		"\n%d new influencer(s) discovered out of %d result(s) checked.\n",
		newCount,
		len(results),
	)
}
