package storage

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
)

type Paths struct {
	AppDir          string
	GitHubTokenPath string
}

func NewPaths() Paths {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Warn("UserHomeDir failed, using current directory", "error", err)
		home = "."
	}
	appDir := filepath.Join(home, ".local", "share", "copilot-api")
	return Paths{
		AppDir:          appDir,
		GitHubTokenPath: filepath.Join(appDir, "github_token"),
	}
}

func EnsurePaths(paths Paths) error {
	if err := os.MkdirAll(paths.AppDir, 0o700); err != nil {
		return err
	}
	return ensureFile(paths.GitHubTokenPath)
}

func ensureFile(path string) error {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
			return err
		}
		return os.Chmod(path, 0o600)
	}
	if err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
