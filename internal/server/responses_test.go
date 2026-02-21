package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/n0madic/go-copilot-api/internal/copilot"
	"github.com/n0madic/go-copilot-api/internal/server"
	"github.com/n0madic/go-copilot-api/internal/sse"
	"github.com/n0madic/go-copilot-api/internal/state"
	"github.com/n0madic/go-copilot-api/internal/types"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestResponsesNonStreamWithPreviousResponseID(t *testing.T) {
	st := newTestState()
	var upstreamPayloads []types.ChatCompletionsPayload
	callNumber := 0

	h := newResponsesTestHandler(t, st, func(req *http.Request) (*http.Response, error) {
		callNumber++
		var payload types.ChatCompletionsPayload
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}
		upstreamPayloads = append(upstreamPayloads, payload)

		message := "A1"
		if callNumber == 2 {
			message = "A2"
		}
		completion := types.ChatCompletionResponse{
			ID:      "chatcmpl_test",
			Object:  "chat.completion",
			Created: 100,
			Model:   "gpt-5-mini",
			Choices: []types.ChoiceNonStreaming{{
				Index: 0,
				Message: types.ResponseMessage{
					Role:    "assistant",
					Content: message,
				},
				FinishReason: "stop",
			}},
			Usage: &types.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
		}
		return jsonHTTPResponse(t, completion), nil
	})

	first := callJSON(t, h, map[string]any{
		"model": "gpt-5-mini",
		"input": "Q1",
	})

	if first.Code != http.StatusOK {
		t.Fatalf("unexpected status for first response: %d body=%s", first.Code, first.Body.String())
	}
	var firstBody types.ResponsesResponse
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if firstBody.ID == "" {
		t.Fatalf("expected response id in first response")
	}

	second := callJSON(t, h, map[string]any{
		"model":                "gpt-5-mini",
		"input":                "Q2",
		"previous_response_id": firstBody.ID,
	})
	if second.Code != http.StatusOK {
		t.Fatalf("unexpected status for second response: %d body=%s", second.Code, second.Body.String())
	}

	if len(upstreamPayloads) != 2 {
		t.Fatalf("expected 2 upstream calls, got %d", len(upstreamPayloads))
	}
	if len(upstreamPayloads[1].Messages) != 3 {
		t.Fatalf("expected 3 messages in second upstream request, got %d", len(upstreamPayloads[1].Messages))
	}
	if upstreamPayloads[1].Messages[0].Role != "user" || upstreamPayloads[1].Messages[0].Content != "Q1" {
		t.Fatalf("unexpected first message: %#v", upstreamPayloads[1].Messages[0])
	}
	if upstreamPayloads[1].Messages[1].Role != "assistant" || upstreamPayloads[1].Messages[1].Content != "A1" {
		t.Fatalf("unexpected second message: %#v", upstreamPayloads[1].Messages[1])
	}
	if upstreamPayloads[1].Messages[2].Role != "user" || upstreamPayloads[1].Messages[2].Content != "Q2" {
		t.Fatalf("unexpected third message: %#v", upstreamPayloads[1].Messages[2])
	}
}

func TestResponsesInvalidPreviousResponseID(t *testing.T) {
	h := newResponsesTestHandler(t, newTestState(), func(req *http.Request) (*http.Response, error) {
		t.Fatalf("upstream should not be called for invalid previous_response_id")
		return nil, nil
	})

	resp := callJSON(t, h, map[string]any{
		"model":                "gpt-5-mini",
		"input":                "Q2",
		"previous_response_id": "resp_missing",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "invalid_previous_response_id" {
		t.Fatalf("unexpected error payload: %#v", errObj)
	}
}

func TestResponsesResolvesModelAlias(t *testing.T) {
	st := newTestState()
	st.SetModels(&types.ModelsResponse{Data: []types.Model{{ID: "claude-sonnet-4.6"}}})

	capturedModel := ""
	h := newResponsesTestHandler(t, st, func(req *http.Request) (*http.Response, error) {
		var payload types.ChatCompletionsPayload
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}
		capturedModel = payload.Model
		completion := types.ChatCompletionResponse{
			ID:      "chatcmpl_test",
			Object:  "chat.completion",
			Created: 100,
			Model:   payload.Model,
			Choices: []types.ChoiceNonStreaming{{
				Index: 0,
				Message: types.ResponseMessage{
					Role:    "assistant",
					Content: "ok",
				},
				FinishReason: "stop",
			}},
		}
		return jsonHTTPResponse(t, completion), nil
	})

	resp := callJSON(t, h, map[string]any{
		"model": "claude-sonnet-4-6",
		"input": "hello",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedModel != "claude-sonnet-4.6" {
		t.Fatalf("expected resolved model claude-sonnet-4.6, got %q", capturedModel)
	}
}

func TestResponsesStreamEvents(t *testing.T) {
	st := newTestState()
	h := newResponsesTestHandler(t, st, func(req *http.Request) (*http.Response, error) {
		chunk1 := types.ChatCompletionChunk{
			ID:      "chatcmpl_stream",
			Object:  "chat.completion.chunk",
			Created: 200,
			Model:   "gpt-5-mini",
			Choices: []types.ChoiceChunk{{
				Index: 0,
				Delta: types.Delta{Content: strPtr("Hel")},
			}},
		}
		chunk2 := types.ChatCompletionChunk{
			ID:      "chatcmpl_stream",
			Object:  "chat.completion.chunk",
			Created: 201,
			Model:   "gpt-5-mini",
			Choices: []types.ChoiceChunk{{
				Index:        0,
				Delta:        types.Delta{Content: strPtr("lo")},
				FinishReason: strPtr("stop"),
			}},
			Usage: &types.Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5},
		}
		body := sseBody(t, chunk1, chunk2)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	stream := true
	resp := callJSON(t, h, map[string]any{
		"model":  "gpt-5-mini",
		"input":  "Hello",
		"stream": stream,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", resp.Code, resp.Body.String())
	}

	events := make([]sse.Event, 0)
	if err := sse.ReadEvents(strings.NewReader(resp.Body.String()), func(ev sse.Event) error {
		events = append(events, ev)
		return nil
	}); err != nil {
		t.Fatalf("parse response SSE: %v", err)
	}

	names := make([]string, 0, len(events))
	for _, ev := range events {
		names = append(names, ev.Event)
	}
	mustContain(t, names, "response.created")
	mustContain(t, names, "response.output_text.delta")
	mustContain(t, names, "response.completed")

	var completed map[string]any
	for _, ev := range events {
		if ev.Event != "response.completed" {
			continue
		}
		if err := json.Unmarshal([]byte(ev.Data), &completed); err != nil {
			t.Fatalf("decode completed event: %v", err)
		}
		break
	}
	responseObj, _ := completed["response"].(map[string]any)
	if responseObj["output_text"] != "Hello" {
		t.Fatalf("unexpected output_text: %#v", responseObj["output_text"])
	}
}

func newResponsesTestHandler(t *testing.T, st *state.State, rt roundTripFunc) http.Handler {
	t.Helper()
	if st == nil {
		st = newTestState()
	}
	if st.CopilotToken() == "" {
		st.SetCopilotToken("test-token")
	}
	if st.VSCodeVersion() == "" {
		st.SetVSCodeVersion("1.0.0")
	}
	if st.AccountType() == "" {
		st.SetAccountType("individual")
	}

	copilotClient := copilot.NewClient(&http.Client{Transport: rt}, st)
	srv := server.New(st, copilotClient, nil)
	return srv.Handler()
}

func newTestState() *state.State {
	st := state.New()
	st.SetAccountType("individual")
	st.SetVSCodeVersion("1.0.0")
	st.SetCopilotToken("test-token")
	return st
}

func callJSON(t *testing.T, h http.Handler, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(b))
	req.Header.Set("content-type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func jsonHTTPResponse(t *testing.T, payload any) *http.Response {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(b)),
	}
}

func sseBody(t *testing.T, chunks ...types.ChatCompletionChunk) string {
	t.Helper()
	var builder strings.Builder
	for _, chunk := range chunks {
		b, err := json.Marshal(chunk)
		if err != nil {
			t.Fatalf("marshal chunk: %v", err)
		}
		builder.WriteString("data: ")
		builder.Write(b)
		builder.WriteString("\n\n")
	}
	builder.WriteString("data: [DONE]\n\n")
	return builder.String()
}

func mustContain(t *testing.T, values []string, target string) {
	t.Helper()
	for _, value := range values {
		if value == target {
			return
		}
	}
	t.Fatalf("expected %q in %v", target, values)
}

func strPtr(v string) *string {
	return &v
}
