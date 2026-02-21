package copilot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/n0madic/go-copilot-api/internal/api"
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

func (c *Client) CreateChatCompletions(ctx context.Context, payload types.ChatCompletionsPayload) (*http.Response, error) {
	if c.state.CopilotToken() == "" {
		return nil, errors.New("copilot token not found")
	}

	enableVision := hasVision(payload)
	isAgentCall := false
	for _, msg := range payload.Messages {
		if msg.Role == "assistant" || msg.Role == "tool" {
			isAgentCall = true
			break
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api.CopilotBaseURL(c.state)+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	headers := api.CopilotHeaders(c.state, enableVision)
	if isAgentCall {
		headers["X-Initiator"] = "agent"
	} else {
		headers["X-Initiator"] = "user"
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, api.NewHTTPError("failed to create chat completions", resp)
	}
	return resp, nil
}

func hasVision(payload types.ChatCompletionsPayload) bool {
	for _, msg := range payload.Messages {
		parts, ok := msg.Content.([]any)
		if !ok {
			continue
		}
		for _, p := range parts {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			typeValue, _ := pm["type"].(string)
			if typeValue == "image_url" {
				return true
			}
		}
	}
	return false
}

func (c *Client) GetModels(ctx context.Context) (types.ModelsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api.CopilotBaseURL(c.state)+"/models", nil)
	if err != nil {
		return types.ModelsResponse{}, err
	}
	for k, v := range api.CopilotHeaders(c.state, false) {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return types.ModelsResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return types.ModelsResponse{}, api.NewHTTPError("failed to get models", resp)
	}
	var out types.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return types.ModelsResponse{}, err
	}
	return out, nil
}

func (c *Client) CreateEmbeddings(ctx context.Context, payload types.EmbeddingRequest) (types.EmbeddingResponse, error) {
	if c.state.CopilotToken() == "" {
		return types.EmbeddingResponse{}, errors.New("copilot token not found")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return types.EmbeddingResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api.CopilotBaseURL(c.state)+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return types.EmbeddingResponse{}, err
	}
	for k, v := range api.CopilotHeaders(c.state, false) {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return types.EmbeddingResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return types.EmbeddingResponse{}, api.NewHTTPError("failed to create embeddings", resp)
	}
	var out types.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return types.EmbeddingResponse{}, err
	}
	return out, nil
}
