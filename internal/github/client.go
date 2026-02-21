package github

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/n0madic/go-copilot-api/internal/api"
	"github.com/n0madic/go-copilot-api/internal/config"
	"github.com/n0madic/go-copilot-api/internal/state"
	"github.com/n0madic/go-copilot-api/internal/types"
)

type Client struct {
	http  *http.Client
	state *state.State
}

func NewClient(httpClient *http.Client, st *state.State) *Client {
	return &Client{http: httpClient, state: st}
}

func (c *Client) GetDeviceCode(ctx context.Context) (types.DeviceCodeResponse, error) {
	payload := map[string]string{
		"client_id": config.GitHubClientID,
		"scope":     config.GitHubAppScopes,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return types.DeviceCodeResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.GitHubBaseURL+"/login/device/code", bytes.NewReader(body))
	if err != nil {
		return types.DeviceCodeResponse{}, err
	}
	for k, v := range api.StandardHeaders() {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return types.DeviceCodeResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return types.DeviceCodeResponse{}, api.NewHTTPError("failed to get device code", resp)
	}
	var out types.DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return types.DeviceCodeResponse{}, err
	}
	return out, nil
}

func (c *Client) PollAccessToken(ctx context.Context, dc types.DeviceCodeResponse) (string, error) {
	sleepDuration := time.Duration(dc.Interval+1) * time.Second
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		payload := map[string]string{
			"client_id":   config.GitHubClientID,
			"device_code": dc.DeviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.GitHubBaseURL+"/login/oauth/access_token", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		for k, v := range api.StandardHeaders() {
			req.Header.Set(k, v)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return "", err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(sleepDuration):
			}
			continue
		}
		var out types.AccessTokenResponse
		err = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		if err != nil {
			return "", err
		}
		if out.AccessToken != "" {
			return out.AccessToken, nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(sleepDuration):
		}
	}
}

func (c *Client) GetCopilotToken(ctx context.Context) (types.GetCopilotTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.GitHubAPIBaseURL+"/copilot_internal/v2/token", nil)
	if err != nil {
		return types.GetCopilotTokenResponse{}, err
	}
	for k, v := range api.GitHubHeaders(c.state) {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return types.GetCopilotTokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return types.GetCopilotTokenResponse{}, api.NewHTTPError("failed to get copilot token", resp)
	}
	var out types.GetCopilotTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return types.GetCopilotTokenResponse{}, err
	}
	return out, nil
}

func (c *Client) GetGitHubUser(ctx context.Context) (types.GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.GitHubAPIBaseURL+"/user", nil)
	if err != nil {
		return types.GitHubUser{}, err
	}
	req.Header.Set("Authorization", "token "+c.state.GitHubToken())
	for k, v := range api.StandardHeaders() {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return types.GitHubUser{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return types.GitHubUser{}, api.NewHTTPError("failed to get github user", resp)
	}
	var out types.GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return types.GitHubUser{}, err
	}
	return out, nil
}

func (c *Client) GetCopilotUsage(ctx context.Context) (types.CopilotUsageResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.GitHubAPIBaseURL+"/copilot_internal/user", nil)
	if err != nil {
		return types.CopilotUsageResponse{}, err
	}
	for k, v := range api.GitHubHeaders(c.state) {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return types.CopilotUsageResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return types.CopilotUsageResponse{}, api.NewHTTPError("failed to get copilot usage", resp)
	}
	var out types.CopilotUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return types.CopilotUsageResponse{}, err
	}
	return out, nil
}
