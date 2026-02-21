package responses

import (
	"testing"

	"github.com/n0madic/go-copilot-api/internal/types"
)

func TestBuildChatCompletionsPayloadStringInput(t *testing.T) {
	stream := true
	temperature := 0.2
	maxOut := 128
	req := types.ResponsesRequest{
		Model:           "gpt-5-mini",
		Input:           "hello",
		Stream:          &stream,
		Temperature:     &temperature,
		MaxOutputTokens: &maxOut,
		Tools: []types.ResponsesTool{{
			Type:        "function",
			Name:        "get_weather",
			Description: "Fetch weather",
			Parameters: map[string]any{
				"type": "object",
			},
		}},
	}

	payload, err := BuildChatCompletionsPayload(req, nil)
	if err != nil {
		t.Fatalf("BuildChatCompletionsPayload returned error: %v", err)
	}

	if payload.Model != "gpt-5-mini" {
		t.Fatalf("unexpected model: %q", payload.Model)
	}
	if payload.Stream == nil || !*payload.Stream {
		t.Fatalf("expected stream=true")
	}
	if payload.MaxTokens == nil || *payload.MaxTokens != 128 {
		t.Fatalf("unexpected max tokens: %v", payload.MaxTokens)
	}
	if len(payload.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(payload.Messages))
	}
	if payload.Messages[0].Role != "user" || payload.Messages[0].Content != "hello" {
		t.Fatalf("unexpected first message: %#v", payload.Messages[0])
	}
	if len(payload.Tools) != 1 || payload.Tools[0].Function.Name != "get_weather" {
		t.Fatalf("unexpected tool translation: %#v", payload.Tools)
	}
}

func TestBuildChatCompletionsPayloadFunctionCallOutput(t *testing.T) {
	req := types.ResponsesRequest{
		Model: "gpt-5-mini",
		Input: []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": "run tool"},
				},
			},
			map[string]any{
				"type":    "function_call_output",
				"call_id": "call_123",
				"output": map[string]any{
					"ok": true,
				},
			},
		},
	}

	history := []types.Message{{
		Role:    "assistant",
		Content: "",
		ToolCalls: []types.ToolCall{{
			ID:   "call_123",
			Type: "function",
			Function: types.ToolCallFn{
				Name:      "my_tool",
				Arguments: `{"x":1}`,
			},
		}},
	}}

	payload, err := BuildChatCompletionsPayload(req, history)
	if err != nil {
		t.Fatalf("BuildChatCompletionsPayload returned error: %v", err)
	}

	if len(payload.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(payload.Messages))
	}

	if payload.Messages[0].Role != "assistant" {
		t.Fatalf("expected first role=assistant (history), got %q", payload.Messages[0].Role)
	}
	if payload.Messages[1].Role != "user" {
		t.Fatalf("expected second role=user, got %q", payload.Messages[1].Role)
	}
	if payload.Messages[1].Content != "run tool" {
		t.Fatalf("unexpected user content: %#v", payload.Messages[1].Content)
	}

	if payload.Messages[2].Role != "tool" {
		t.Fatalf("expected third role=tool, got %q", payload.Messages[2].Role)
	}
	if payload.Messages[2].ToolCallID != "call_123" {
		t.Fatalf("unexpected tool_call_id: %q", payload.Messages[2].ToolCallID)
	}
	if payload.Messages[2].Content != `{"ok":true}` {
		t.Fatalf("unexpected tool content: %q", payload.Messages[2].Content)
	}
}

func TestCompletionToResponseIncludesFunctionCalls(t *testing.T) {
	completion := types.ChatCompletionResponse{
		ID:      "chatcmpl_1",
		Object:  "chat.completion",
		Created: 123,
		Model:   "gpt-5-mini",
		Choices: []types.ChoiceNonStreaming{{
			Index: 0,
			Message: types.ResponseMessage{
				Role:    "assistant",
				Content: "I will call a tool",
				ToolCalls: []types.ToolCall{{
					ID:   "call_1",
					Type: "function",
					Function: types.ToolCallFn{
						Name:      "get_weather",
						Arguments: `{"city":"Berlin"}`,
					},
				}},
			},
			FinishReason: "tool_calls",
		}},
		Usage: &types.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	req := types.ResponsesRequest{Model: "gpt-5-mini"}
	response, assistant := CompletionToResponse(req, completion, "resp_test")

	if response.ID != "resp_test" {
		t.Fatalf("unexpected response id: %q", response.ID)
	}
	if response.Object != "response" || response.Status != "completed" {
		t.Fatalf("unexpected envelope: %#v", response)
	}
	if response.OutputText != "I will call a tool" {
		t.Fatalf("unexpected output_text: %q", response.OutputText)
	}
	if len(response.Output) != 2 {
		t.Fatalf("expected 2 output items, got %d", len(response.Output))
	}
	if response.Output[1].Type != "function_call" {
		t.Fatalf("expected second output item to be function_call, got %q", response.Output[1].Type)
	}
	if response.Usage == nil || response.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected usage: %#v", response.Usage)
	}
	if len(assistant.ToolCalls) != 1 || assistant.ToolCalls[0].ID != "call_1" {
		t.Fatalf("unexpected assistant message tool calls: %#v", assistant.ToolCalls)
	}
}

func TestCompletionToResponseNormalizesMissingToolCallID(t *testing.T) {
	completion := types.ChatCompletionResponse{
		ID:      "chatcmpl_missing_id",
		Object:  "chat.completion",
		Created: 123,
		Model:   "gpt-5-mini",
		Choices: []types.ChoiceNonStreaming{{
			Index: 0,
			Message: types.ResponseMessage{
				Role:    "assistant",
				Content: nil,
				ToolCalls: []types.ToolCall{{
					ID:   "",
					Type: "function",
					Function: types.ToolCallFn{
						Name:      "get_weather",
						Arguments: `{"city":"Paris"}`,
					},
				}},
			},
			FinishReason: "tool_calls",
		}},
	}

	response, assistant := CompletionToResponse(types.ResponsesRequest{Model: "gpt-5-mini"}, completion, "resp_norm")
	if len(response.Output) != 1 {
		t.Fatalf("expected single function_call output item, got %d", len(response.Output))
	}
	if response.Output[0].Type != "function_call" {
		t.Fatalf("unexpected output type: %q", response.Output[0].Type)
	}
	if response.Output[0].CallID == "" {
		t.Fatalf("expected generated call_id in response output")
	}
	if len(assistant.ToolCalls) != 1 {
		t.Fatalf("expected one tool call in assistant message")
	}
	if assistant.ToolCalls[0].ID != response.Output[0].CallID {
		t.Fatalf("assistant tool_call id and response call_id must match, got %q vs %q", assistant.ToolCalls[0].ID, response.Output[0].CallID)
	}
	if assistant.Content != "" {
		t.Fatalf("expected assistant content to be empty string, got %#v", assistant.Content)
	}
}

func TestBuildChatCompletionsPayloadRejectsUnmatchedToolMessage(t *testing.T) {
	req := types.ResponsesRequest{
		Model: "gpt-5-mini",
		Input: []any{
			map[string]any{
				"type":    "function_call_output",
				"call_id": "call_missing",
				"output":  "ok",
			},
		},
	}

	_, err := BuildChatCompletionsPayload(req, nil)
	if err == nil {
		t.Fatalf("expected validation error for unmatched tool message")
	}
	if got := err.Error(); got == "" {
		t.Fatalf("expected non-empty error")
	}
}
