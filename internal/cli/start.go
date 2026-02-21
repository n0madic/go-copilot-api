package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/n0madic/go-copilot-api/internal/config"
	"github.com/n0madic/go-copilot-api/internal/copilot"
	"github.com/n0madic/go-copilot-api/internal/github"
	"github.com/n0madic/go-copilot-api/internal/server"
	"github.com/n0madic/go-copilot-api/internal/state"
	"github.com/n0madic/go-copilot-api/internal/storage"
	"github.com/n0madic/go-copilot-api/internal/types"
	"github.com/n0madic/go-copilot-api/internal/util"
)

func runStart(args []string) int {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	var (
		finalPort        int
		bindAddr         string
		verbose          bool
		account          string
		rateLimitRaw     string
		waitEnabled      bool
		gitHubTokenValue string
		claudeEnabled    bool
		showToken        bool
		proxyEnv         bool
	)
	fs.IntVar(&finalPort, "port", config.DefaultPort, "Port to listen on")
	fs.StringVar(&bindAddr, "bind", "127.0.0.1", "Bind address (e.g. 127.0.0.1, 0.0.0.0, ::)")
	fs.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	fs.StringVar(&account, "account-type", config.DefaultAccountType, "Account type: individual|business|enterprise")
	fs.StringVar(&rateLimitRaw, "rate-limit", "", "Rate limit in seconds between requests")
	fs.BoolVar(&waitEnabled, "wait", false, "Wait instead of error when rate limit is hit")
	fs.StringVar(&gitHubTokenValue, "github-token", "", "Provide GitHub token directly")
	fs.BoolVar(&claudeEnabled, "claude-code", false, "Generate Claude Code launch command")
	fs.BoolVar(&showToken, "show-token", false, "Show GitHub and Copilot token")
	fs.BoolVar(&proxyEnv, "proxy-env", false, "Initialize proxy from environment")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if verbose {
		setLogger(slog.LevelDebug)
		slog.Info("Verbose logging enabled")
	} else {
		setLogger(slog.LevelInfo)
	}

	if account != "individual" {
		slog.Info("Using non-individual account", "account_type", account)
	}

	rateLimitRaw = strings.TrimSpace(rateLimitRaw)
	var rateLimitSeconds *int
	if rateLimitRaw != "" {
		parsed, err := strconv.Atoi(rateLimitRaw)
		if err != nil || parsed <= 0 {
			fmt.Fprintln(os.Stderr, "invalid --rate-limit value")
			return 1
		}
		rateLimitSeconds = &parsed
	}

	st := state.New()
	st.SetAccountType(account)
	st.SetRateLimitWait(waitEnabled)
	st.SetShowToken(showToken)
	st.SetRateLimitSeconds(rateLimitSeconds)

	paths := storage.NewPaths()
	if err := storage.EnsurePaths(paths); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ensure paths: %v\n", err)
		return 1
	}

	httpClient := newHTTPClient(proxyEnv)
	vscodeVersion := util.GetVSCodeVersion(httpClient, paths.AppDir)
	st.SetVSCodeVersion(vscodeVersion)
	slog.Info("Using VSCode version", "version", vscodeVersion)

	ghClient := github.NewClient(httpClient, st)
	tokenManager := github.NewTokenManager(ghClient, st, paths)

	// authCtx is used only for the initial auth handshake (5 min timeout).
	// serverCtx is used for the server lifetime (token refresh loop cancellation).
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	authCtx, authCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer authCancel()

	if gitHubTokenValue != "" {
		st.SetGitHubToken(gitHubTokenValue)
		slog.Info("Using provided GitHub token")
	} else {
		if err := tokenManager.SetupGitHubToken(authCtx, github.SetupGitHubTokenOptions{}); err != nil {
			fmt.Fprintf(os.Stderr, "auth failed: %v\n", err)
			return 1
		}
	}

	if err := tokenManager.SetupCopilotToken(serverCtx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup copilot token: %v\n", err)
		return 1
	}

	copilotClient := copilot.NewClient(httpClient, st)
	if err := copilot.CacheModels(authCtx, copilotClient, st); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load models: %v\n", err)
		return 1
	}

	if models := st.Models(); models != nil {
		printAvailableModels(models.Data)
	}

	serverURL := fmt.Sprintf("http://localhost:%d", finalPort)
	if claudeEnabled {
		if err := generateClaudeCodeCommand(st, serverURL); err != nil {
			slog.Warn("failed to generate claude code command", "error", err)
		}
	}

	srv := server.New(st, copilotClient, ghClient)
	httpServer := &http.Server{
		Addr:              net.JoinHostPort(bindAddr, strconv.Itoa(finalPort)),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
		return 1
	}
	return 0
}

func printAvailableModels(models []types.Model) {
	if len(models) == 0 {
		fmt.Println("Available models: none")
		return
	}

	seen := make(map[string]struct{}, len(models))
	uniqueIDs := make([]string, 0, len(models))
	for _, model := range models {
		if _, ok := seen[model.ID]; ok {
			continue
		}
		seen[model.ID] = struct{}{}
		uniqueIDs = append(uniqueIDs, model.ID)
	}

	sort.Strings(uniqueIDs)

	fmt.Printf("Available models (%d):\n", len(uniqueIDs))
	for _, id := range uniqueIDs {
		fmt.Printf("  - %s\n", id)
	}
}
