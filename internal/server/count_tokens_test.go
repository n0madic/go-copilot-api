package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/n0madic/go-copilot-api/internal/server"
	"github.com/n0madic/go-copilot-api/internal/state"
	"github.com/n0madic/go-copilot-api/internal/types"
)

func TestCountTokensAddsToolOverheadForClaude(t *testing.T) {
	st := state.New()
	st.SetModels(&types.ModelsResponse{Data: []types.Model{{
		ID: "claude-sonnet-4",
		Capabilities: types.ModelCapabilities{
			Tokenizer: "o200k_base",
			Limits:    types.ModelLimits{},
		},
	}}})
	srv := server.New(st, nil, nil)
	h := srv.Handler()

	basePayload := map[string]any{
		"model": "claude-sonnet-4",
		"messages": []any{
			map[string]any{"role": "user", "content": "hello"},
		},
		"max_tokens": 100,
	}
	withTools := map[string]any{
		"model": "claude-sonnet-4",
		"messages": []any{
			map[string]any{"role": "user", "content": "hello"},
		},
		"tools": []any{
			map[string]any{"name": "my_tool", "input_schema": map[string]any{"type": "object"}},
		},
		"max_tokens": 100,
	}

	base := callCountTokens(t, h, basePayload, "claude-code-v1")
	with := callCountTokens(t, h, withTools, "claude-code-v1")
	if with <= base {
		t.Fatalf("expected tools payload count > base, base=%d with=%d", base, with)
	}
}

func TestCountTokensAcceptsClaudeAliasModel(t *testing.T) {
	st := state.New()
	st.SetModels(&types.ModelsResponse{Data: []types.Model{{
		ID: "claude-sonnet-4.6",
		Capabilities: types.ModelCapabilities{
			Tokenizer: "o200k_base",
			Limits:    types.ModelLimits{},
		},
	}}})
	srv := server.New(st, nil, nil)
	h := srv.Handler()

	payload := map[string]any{
		"model": "claude-sonnet-4-6",
		"messages": []any{
			map[string]any{"role": "user", "content": "hello"},
		},
		"max_tokens": 100,
	}

	result := callCountTokens(t, h, payload, "claude-code-v1")
	if result <= 1 {
		t.Fatalf("expected resolved model token count > 1, got %d", result)
	}
}

func callCountTokens(t *testing.T, h http.Handler, payload map[string]any, beta string) int {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", bytes.NewReader(b))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("anthropic-beta", beta)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}
	var out map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return out["input_tokens"]
}
