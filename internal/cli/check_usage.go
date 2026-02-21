package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/n0madic/go-copilot-api/internal/github"
	"github.com/n0madic/go-copilot-api/internal/state"
	"github.com/n0madic/go-copilot-api/internal/storage"
	"github.com/n0madic/go-copilot-api/internal/util"
)

func runCheckUsage(args []string) int {
	fs := flag.NewFlagSet("check-usage", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	jsonOut := fs.Bool("json", false, "Output full usage response as pretty JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *jsonOut {
		setLogger(slog.LevelError)
	} else {
		setLogger(slog.LevelInfo)
	}

	st := state.New()
	paths := storage.NewPaths()
	if err := storage.EnsurePaths(paths); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ensure paths: %v\n", err)
		return 1
	}
	client := newHTTPClient(false)
	st.SetVSCodeVersion(util.GetVSCodeVersion(client, paths.AppDir))
	ghClient := github.NewClient(client, st)
	tm := github.NewTokenManager(ghClient, st, paths)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := tm.SetupGitHubToken(ctx, github.SetupGitHubTokenOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup github token: %v\n", err)
		return 1
	}
	usage, err := ghClient.GetCopilotUsage(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch usage: %v\n", err)
		return 1
	}
	if *jsonOut {
		b, err := json.MarshalIndent(usage, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to render usage as json: %v\n", err)
			return 1
		}
		fmt.Println(string(b))
		return 0
	}
	premium := usage.QuotaSnapshots.PremiumInteractions
	premiumLine := util.HumanizeQuotaUsed(premium.Entitlement, premium.Remaining, premium.PercentRemaining)
	chat := usage.QuotaSnapshots.Chat
	chatLine := util.HumanizeQuotaUsed(chat.Entitlement, chat.Remaining, chat.PercentRemaining)
	completions := usage.QuotaSnapshots.Completions
	completionsLine := util.HumanizeQuotaUsed(completions.Entitlement, completions.Remaining, completions.PercentRemaining)
	fmt.Printf("Copilot Usage (plan: %s)\n", usage.CopilotPlan)
	fmt.Printf("Quota resets: %s\n\n", usage.QuotaResetDate)
	fmt.Printf("Quotas:\n")
	fmt.Printf("  Premium: %s\n", premiumLine)
	fmt.Printf("  Chat: %s\n", chatLine)
	fmt.Printf("  Completions: %s\n", completionsLine)
	return 0
}
