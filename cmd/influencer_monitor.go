package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/spf13/cobra"
)

var influencerMonitorUsernames string

var influencerMonitorCmd = &cobra.Command{
	Use:   "influencer-monitor",
	Short: "Check tracked TikTok influencers for new uploads and print the result",
	Long: "MVP command for the misinformation-monitoring pipeline: fetches each\n" +
		"influencer's latest videos via Apify, stores/updates their profile and\n" +
		"video metadata (likes, views, comments, shares, collects), recomputes\n" +
		"their average upload interval, and prints a summary to the terminal.\n" +
		"Meant to run on a schedule (e.g. every 24h) in production; for now it's\n" +
		"a plain CLI command so the pipeline can be demoed end-to-end.",
	Run: runInfluencerMonitor,
}

func init() {
	influencerMonitorCmd.Flags().StringVar(
		&influencerMonitorUsernames,
		"usernames",
		"",
		"comma-separated TikTok usernames to check (default: every influencer already tracked, "+
			"falling back to INFLUENCER_SEED_USERNAMES / dummy seed list if none tracked yet)",
	)
}

func runInfluencerMonitor(cmd *cobra.Command, _ []string) {
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

	usernames, err := resolveMonitorUsernames(ctx, cmd, influencerRepo, config)
	if err != nil {
		cmd.PrintErrf("Failed to resolve usernames: %s\n", err.Error())
		os.Exit(1)
		return
	}

	cmd.Printf("Checking %d influencer(s)...\n\n", len(usernames))

	results, err := monitorService.MonitorInfluencers(
		ctx,
		usernames,
		config.InfluencerMonitorConfig.ResultsPerProfile,
	)
	if err != nil {
		cmd.PrintErrf("Monitor run failed: %s\n", err.Error())
		os.Exit(1)
		return
	}

	printMonitorResults(cmd, results)
}

func resolveMonitorUsernames(
	ctx context.Context,
	cmd *cobra.Command,
	influencerRepo data.InfluencerRepoer,
	config *internal.Config,
) ([]string, error) {
	if influencerMonitorUsernames != "" {
		return splitAndTrim(influencerMonitorUsernames), nil
	}

	tracked, err := influencerRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	if len(tracked) > 0 {
		usernames := make([]string, 0, len(tracked))
		for _, inf := range tracked {
			usernames = append(usernames, inf.Username)
		}
		return usernames, nil
	}

	cmd.Println(
		"No influencers tracked yet — using the dummy seed list from INFLUENCER_SEED_USERNAMES.",
	)
	return config.InfluencerMonitorConfig.SeedUsernames, nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func printMonitorResults(cmd *cobra.Command, results []service.InfluencerMonitorResult) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "USERNAME\tSTATUS\tVIDEOS SEEN\tNEW VIDEOS\tAVG UPLOAD INTERVAL")

	for _, r := range results {
		status := "ok"
		switch {
		case r.Error != "":
			status = "error: " + r.Error
		case !r.ProfileFound:
			status = "profile not found"
		}

		interval := "-"
		if r.AvgUploadInterval != nil {
			interval = formatUploadInterval(*r.AvgUploadInterval)
		}

		fmt.Fprintf(
			w,
			"%s\t%s\t%d\t%d\t%s\n",
			r.Username,
			status,
			r.TotalVideosSeen,
			len(r.NewVideos),
			interval,
		)
	}
	_ = w.Flush()

	for _, r := range results {
		if len(r.NewVideos) == 0 {
			continue
		}
		cmd.Printf("\nNew videos for @%s:\n", r.Username)
		for _, v := range r.NewVideos {
			cmd.Printf(
				"  - %s | %s\n    %s\n",
				v.CreatedAt.Format(time.RFC3339),
				truncateText(v.Caption, 60),
				v.VideoURL,
			)
		}
	}
}

func formatUploadInterval(d time.Duration) string {
	hours := d.Hours()
	if hours < 24 {
		return fmt.Sprintf("%.1fh", hours)
	}
	return fmt.Sprintf("%.1fd", hours/24)
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
