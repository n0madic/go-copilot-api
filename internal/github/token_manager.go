package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/n0madic/go-copilot-api/internal/state"
	"github.com/n0madic/go-copilot-api/internal/storage"
)

type TokenManager struct {
	client *Client
	state  *state.State
	paths  storage.Paths
}

func NewTokenManager(client *Client, st *state.State, paths storage.Paths) *TokenManager {
	return &TokenManager{client: client, state: st, paths: paths}
}

type SetupGitHubTokenOptions struct {
	Force bool
}

func (m *TokenManager) SetupGitHubToken(ctx context.Context, options SetupGitHubTokenOptions) error {
	token, err := os.ReadFile(m.paths.GitHubTokenPath)
	if err == nil && strings.TrimSpace(string(token)) != "" && !options.Force {
		readToken := strings.TrimSpace(string(token))
		m.state.SetGitHubToken(readToken)
		if m.state.ShowToken() {
			slog.Info("GitHub token", "token", readToken)
		}
		return m.logUser(ctx)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	slog.Info("Not logged in, getting new access token")
	dc, err := m.client.GetDeviceCode(ctx)
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Please enter the code in %s", dc.VerificationURI), "code", dc.UserCode)
	accessToken, err := m.client.PollAccessToken(ctx, dc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(m.paths.GitHubTokenPath, []byte(accessToken), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(m.paths.GitHubTokenPath, 0o600); err != nil {
		return err
	}
	m.state.SetGitHubToken(accessToken)
	if m.state.ShowToken() {
		slog.Info("GitHub token", "token", accessToken)
	}
	return m.logUser(ctx)
}

// SetupCopilotToken fetches the initial Copilot token and starts the background
// refresh loop.  The provided ctx is used to cancel the refresh loop when the
// server shuts down; it should be a long-lived context (e.g. server lifetime),
// not the short auth timeout context.
func (m *TokenManager) SetupCopilotToken(ctx context.Context) error {
	resp, err := m.client.GetCopilotToken(ctx)
	if err != nil {
		return err
	}
	m.state.SetCopilotToken(resp.Token)
	slog.Debug("GitHub Copilot token fetched successfully")
	if m.state.ShowToken() {
		slog.Info("Copilot token", "token", resp.Token)
	}

	refreshIn := max(resp.RefreshIn-60, 10)
	go m.refreshLoop(ctx, refreshIn)
	return nil
}

func (m *TokenManager) refreshLoop(ctx context.Context, refreshInSeconds int) {
	ticker := time.NewTicker(time.Duration(refreshInSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Debug("Copilot token refresh loop stopped")
			return
		case <-ticker.C:
			slog.Debug("Refreshing Copilot token")
			refreshCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			resp, err := m.client.GetCopilotToken(refreshCtx)
			cancel()
			if err != nil {
				slog.Error("Failed to refresh Copilot token", "error", err)
				continue
			}
			m.state.SetCopilotToken(resp.Token)
			if m.state.ShowToken() {
				slog.Info("Refreshed Copilot token", "token", resp.Token)
			}
		}
	}
}

func (m *TokenManager) logUser(ctx context.Context) error {
	user, err := m.client.GetGitHubUser(ctx)
	if err != nil {
		return err
	}
	slog.Info("Logged in", "user", user.Login)
	return nil
}
