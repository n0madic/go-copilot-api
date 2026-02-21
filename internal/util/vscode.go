package util

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/n0madic/go-copilot-api/internal/config"
)

var pkgverRE = regexp.MustCompile(`pkgver=([0-9.]+)`)

const vscodeCacheFile = "vscode_version"
const vscodeCacheTTL = 24 * time.Hour

// GetVSCodeVersion returns the current VSCode version.
// It checks a disk cache (in cacheDir) with a 24-hour TTL before hitting the
// remote AUR PKGBUILD.  On any error it falls back to config.FallbackVSCode.
func GetVSCodeVersion(client *http.Client, cacheDir string) string {
	if cacheDir != "" {
		if v := readVSCodeCache(cacheDir); v != "" {
			return v
		}
	}

	v := fetchVSCodeVersion(client)
	if cacheDir != "" && v != config.FallbackVSCode {
		if err := writeVSCodeCache(cacheDir, v); err != nil {
			slog.Warn("failed to write vscode version cache", "error", err)
		}
	}
	return v
}

func readVSCodeCache(cacheDir string) string {
	path := filepath.Join(cacheDir, vscodeCacheFile)
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	if time.Since(info.ModTime()) > vscodeCacheTTL {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(data))
	if v == "" {
		return ""
	}
	return v
}

func writeVSCodeCache(cacheDir string, version string) error {
	path := filepath.Join(cacheDir, vscodeCacheFile)
	return os.WriteFile(path, []byte(version), 0o600)
}

func fetchVSCodeVersion(client *http.Client) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.VSCodeVersionSource, nil)
	if err != nil {
		return config.FallbackVSCode
	}
	resp, err := client.Do(req)
	if err != nil {
		return config.FallbackVSCode
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return config.FallbackVSCode
	}
	m := pkgverRE.FindStringSubmatch(string(body))
	if len(m) > 1 {
		return m[1]
	}
	return config.FallbackVSCode
}
