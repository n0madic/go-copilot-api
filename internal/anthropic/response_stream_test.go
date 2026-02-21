package anthropic_test

import (
	"testing"

	"github.com/n0madic/go-copilot-api/internal/anthropic"
	"github.com/n0madic/go-copilot-api/internal/types"
)

func isValidAnthropicResponse(payload types.AnthropicResponse) bool {
	if payload.ID == "" || payload.Type != "message" || payload.Role != "assistant" || payload.Model == "" {
		return false
	}
	if payload.Usage.InputTokens < 0 || payload.Usage.OutputTokens < 0 {
		return false
	}
	for _, block := range payload.Content {
		typeValue, _ := block["type"].(string)
		if typeValue != "text" && typeValue != "tool_use" {
			return false
		}
	}
	return true
}

func isValidAnthropicStreamEvent(event map[string]any) bool {
	t, _ := event["type"].(string)
	switch t {
	case "message_start", "content_block_start", "content_block_delta", "content_block_stop", "message_delta", "message_stop":
		return true
	default:
		return false
	}
}

func TestTranslateSimpleTextResponse(t *testing.T) {
	openAIResponse := types.ChatCompletionResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "gpt-4o-2024-05-13",
		Choices: []types.ChoiceNonStreaming{{
			Index: 0,
			Message: types.ResponseMessage{
				Role:    "assistant",
				Content: "Hello! How can I help you today?",
			},
			FinishReason: "stop",
			Logprobs:     nil,
		}},
		Usage: &types.Usage{PromptTokens: 9, CompletionTokens: 12, TotalTokens: 21},
	}

	anthropicResponse := anthropic.TranslateToAnthropic(openAIResponse)
	if !isValidAnthropicResponse(anthropicResponse) {
		t.Fatalf("invalid anthropic response: %#v", anthropicResponse)
	}
	if anthropicResponse.ID != "chatcmpl-123" {
		t.Fatalf("unexpected id: %s", anthropicResponse.ID)
	}
	if anthropicResponse.StopReason != "end_turn" {
		t.Fatalf("unexpected stop reason: %#v", anthropicResponse.StopReason)
	}
	if anthropicResponse.Usage.InputTokens != 9 {
		t.Fatalf("unexpected input_tokens: %d", anthropicResponse.Usage.InputTokens)
	}
	if len(anthropicResponse.Content) == 0 || anthropicResponse.Content[0]["type"] != "text" {
		t.Fatalf("expected first content block to be text")
	}
	if anthropicResponse.Content[0]["text"] != "Hello! How can I help you today?" {
		t.Fatalf("unexpected text block value: %#v", anthropicResponse.Content[0]["text"])
	}
}

func TestTranslateResponseWithToolCalls(t *testing.T) {
	openAIResponse := types.ChatCompletionResponse{
		ID:      "chatcmpl-456",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "gpt-4o-2024-05-13",
		Choices: []types.ChoiceNonStreaming{{
			Index: 0,
			Message: types.ResponseMessage{
				Role:    "assistant",
				Content: nil,
				ToolCalls: []types.ToolCall{{
					ID:   "call_abc",
					Type: "function",
					Function: types.ToolCallFn{
						Name:      "get_current_weather",
						Arguments: `{"location": "Boston, MA"}`,
					},
				}},
			},
			FinishReason: "tool_calls",
			Logprobs:     nil,
		}},
		Usage: &types.Usage{PromptTokens: 30, CompletionTokens: 20, TotalTokens: 50},
	}
	anthropicResponse := anthropic.TranslateToAnthropic(openAIResponse)
	if !isValidAnthropicResponse(anthropicResponse) {
		t.Fatalf("invalid anthropic response")
	}
	if anthropicResponse.StopReason != "tool_use" {
		t.Fatalf("unexpected stop reason: %#v", anthropicResponse.StopReason)
	}
	if len(anthropicResponse.Content) == 0 || anthropicResponse.Content[0]["type"] != "tool_use" {
		t.Fatalf("expected tool_use block")
	}
	if anthropicResponse.Content[0]["id"] != "call_abc" {
		t.Fatalf("unexpected tool id: %#v", anthropicResponse.Content[0]["id"])
	}
}

func TestTranslateResponseStoppedDueToLength(t *testing.T) {
	openAIResponse := types.ChatCompletionResponse{
		ID:      "chatcmpl-789",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "gpt-4o-2024-05-13",
		Choices: []types.ChoiceNonStreaming{{
			Index:        0,
			Message:      types.ResponseMessage{Role: "assistant", Content: "This is a very long response that was cut off..."},
			FinishReason: "length",
			Logprobs:     nil,
		}},
		Usage: &types.Usage{PromptTokens: 10, CompletionTokens: 2048, TotalTokens: 2058},
	}
	anthropicResponse := anthropic.TranslateToAnthropic(openAIResponse)
	if !isValidAnthropicResponse(anthropicResponse) {
		t.Fatalf("invalid anthropic response")
	}
	if anthropicResponse.StopReason != "max_tokens" {
		t.Fatalf("unexpected stop reason: %#v", anthropicResponse.StopReason)
	}
}

func TestTranslateSimpleTextStream(t *testing.T) {
	openAIStream := []types.ChatCompletionChunk{
		{
			ID:      "cmpl-1",
			Object:  "chat.completion.chunk",
			Created: 1677652288,
			Model:   "gpt-4o-2024-05-13",
			Choices: []types.ChoiceChunk{{
				Index:        0,
				Delta:        types.Delta{Role: "assistant"},
				FinishReason: nil,
				Logprobs:     nil,
			}},
		},
		{
			ID:      "cmpl-1",
			Object:  "chat.completion.chunk",
			Created: 1677652288,
			Model:   "gpt-4o-2024-05-13",
			Choices: []types.ChoiceChunk{{
				Index:        0,
				Delta:        types.Delta{Content: strPtr("Hello")},
				FinishReason: nil,
				Logprobs:     nil,
			}},
		},
		{
			ID:      "cmpl-1",
			Object:  "chat.completion.chunk",
			Created: 1677652288,
			Model:   "gpt-4o-2024-05-13",
			Choices: []types.ChoiceChunk{{
				Index:        0,
				Delta:        types.Delta{Content: strPtr(" there")},
				FinishReason: nil,
				Logprobs:     nil,
			}},
		},
		{
			ID:      "cmpl-1",
			Object:  "chat.completion.chunk",
			Created: 1677652288,
			Model:   "gpt-4o-2024-05-13",
			Choices: []types.ChoiceChunk{{
				Index:        0,
				Delta:        types.Delta{},
				FinishReason: strPtr("stop"),
				Logprobs:     nil,
			}},
		},
	}

	state := &types.AnthropicStreamState{ToolCalls: map[int]types.ToolCallState{}}
	translated := make([]map[string]any, 0)
	for _, chunk := range openAIStream {
		translated = append(translated, anthropic.TranslateChunkToAnthropicEvents(chunk, state)...)
	}
	for _, event := range translated {
		if !isValidAnthropicStreamEvent(event) {
			t.Fatalf("invalid stream event: %#v", event)
		}
	}
}

func TestTranslateStreamWithToolCalls(t *testing.T) {
	openAIStream := []types.ChatCompletionChunk{
		{
			ID:      "cmpl-2",
			Object:  "chat.completion.chunk",
			Created: 1677652288,
			Model:   "gpt-4o-2024-05-13",
			Choices: []types.ChoiceChunk{{
				Index:        0,
				Delta:        types.Delta{Role: "assistant"},
				FinishReason: nil,
				Logprobs:     nil,
			}},
		},
		{
			ID:      "cmpl-2",
			Object:  "chat.completion.chunk",
			Created: 1677652288,
			Model:   "gpt-4o-2024-05-13",
			Choices: []types.ChoiceChunk{{
				Index: 0,
				Delta: types.Delta{ToolCalls: []types.ToolCallDelta{{Index: 0, ID: "call_xyz", Type: "function", Function: &types.ToolCallFnDelta{Name: "get_weather", Arguments: ""}}}},
			}},
		},
		{
			ID:      "cmpl-2",
			Object:  "chat.completion.chunk",
			Created: 1677652288,
			Model:   "gpt-4o-2024-05-13",
			Choices: []types.ChoiceChunk{{
				Index: 0,
				Delta: types.Delta{ToolCalls: []types.ToolCallDelta{{Index: 0, Function: &types.ToolCallFnDelta{Arguments: `{"loc`}}}},
			}},
		},
		{
			ID:      "cmpl-2",
			Object:  "chat.completion.chunk",
			Created: 1677652288,
			Model:   "gpt-4o-2024-05-13",
			Choices: []types.ChoiceChunk{{
				Index: 0,
				Delta: types.Delta{ToolCalls: []types.ToolCallDelta{{Index: 0, Function: &types.ToolCallFnDelta{Arguments: `ation": "Paris"}`}}}},
			}},
		},
		{
			ID:      "cmpl-2",
			Object:  "chat.completion.chunk",
			Created: 1677652288,
			Model:   "gpt-4o-2024-05-13",
			Choices: []types.ChoiceChunk{{
				Index:        0,
				Delta:        types.Delta{},
				FinishReason: strPtr("tool_calls"),
			}},
		},
	}

	state := &types.AnthropicStreamState{ToolCalls: map[int]types.ToolCallState{}}
	translated := make([]map[string]any, 0)
	for _, chunk := range openAIStream {
		translated = append(translated, anthropic.TranslateChunkToAnthropicEvents(chunk, state)...)
	}
	for _, event := range translated {
		if !isValidAnthropicStreamEvent(event) {
			t.Fatalf("invalid stream event: %#v", event)
		}
	}
}
