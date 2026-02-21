package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/n0madic/go-copilot-api/internal/storage"
)

func runDebug(args []string) int {
	fs := flag.NewFlagSet("debug", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	jsonOut := fs.Bool("json", false, "Output debug information as JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	setLogger(slog.LevelInfo)

	paths := storage.NewPaths()
	tokenExists := false
	if b, err := os.ReadFile(paths.GitHubTokenPath); err == nil && strings.TrimSpace(string(b)) != "" {
		tokenExists = true
	}

	info := map[string]any{
		"runtime": map[string]any{
			"name":     "go",
			"version":  runtimeVersion(),
			"platform": runtime.GOOS,
			"arch":     runtime.GOARCH,
		},
		"paths": map[string]any{
			"APP_DIR":           paths.AppDir,
			"GITHUB_TOKEN_PATH": paths.GitHubTokenPath,
		},
		"tokenExists": tokenExists,
	}

	if *jsonOut {
		b, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(b))
		return 0
	}

	fmt.Printf("copilot-api debug\n\n")
	rt := info["runtime"].(map[string]any)
	fmt.Printf("Runtime: %s %s (%s %s)\n\n", rt["name"], rt["version"], rt["platform"], rt["arch"])
	p := info["paths"].(map[string]any)
	fmt.Printf("Paths:\n- APP_DIR: %s\n- GITHUB_TOKEN_PATH: %s\n\n", p["APP_DIR"], p["GITHUB_TOKEN_PATH"])
	if tokenExists {
		fmt.Println("Token exists: Yes")
	} else {
		fmt.Println("Token exists: No")
	}
	return 0
}
