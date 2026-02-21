package cli

import (
	"context"
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

func runAuth(args []string) int {
	fs := flag.NewFlagSet("auth", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	var verbose bool
	fs.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	showToken := fs.Bool("show-token", false, "Show GitHub token")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if verbose {
		setLogger(slog.LevelDebug)
		slog.Info("Verbose logging enabled")
	} else {
		setLogger(slog.LevelInfo)
	}

	st := state.New()
	st.SetShowToken(*showToken)
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
	if err := tm.SetupGitHubToken(ctx, github.SetupGitHubTokenOptions{Force: true}); err != nil {
		fmt.Fprintf(os.Stderr, "auth failed: %v\n", err)
		return 1
	}
	fmt.Printf("GitHub token written to %s\n", paths.GitHubTokenPath)
	return 0
}
