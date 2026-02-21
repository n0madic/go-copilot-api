package anthropic_test

import (
	"testing"

	"github.com/n0madic/go-copilot-api/internal/anthropic"
	"github.com/n0madic/go-copilot-api/internal/types"
)

func isValidChatCompletionRequest(payload types.ChatCompletionsPayload) bool {
	if payload.Model == "" {
		return false
	}
	if len(payload.Messages) == 0 {
		return false
	}
	for _, m := range payload.Messages {
		if m.Role == "" {
			return false
		}
	}
	return true
}

func TestTranslateMinimalAnthropicPayload(t *testing.T) {
	anthropicPayload := types.AnthropicMessagesPayload{
		Model:     "gpt-4o",
		Messages:  []types.AnthropicMessage{{Role: "user", Content: "Hello!"}},
		MaxTokens: 0,
	}
	openAIPayload := anthropic.TranslateToOpenAI(anthropicPayload)
	if !isValidChatCompletionRequest(openAIPayload) {
		t.Fatalf("translated payload is invalid: %#v", openAIPayload)
	}
}

func TestTranslateComprehensiveAnthropicPayload(t *testing.T) {
	temp := 0.7
	topP := 1.0
	stream := false
	anthropicPayload := types.AnthropicMessagesPayload{
		Model:  "gpt-4o",
		System: "You are a helpful assistant.",
		Messages: []types.AnthropicMessage{
			{Role: "user", Content: "What is the weather like in Boston?"},
			{Role: "assistant", Content: "The weather in Boston is sunny and 75°F."},
		},
		Temperature: tempPtr(temp),
		MaxTokens:   150,
		TopP:        tempPtr(topP),
		Stream:      &stream,
		Metadata:    &types.AnthropicMetadata{UserID: "user-123"},
		Tools: []types.AnthropicTool{{
			Name:        "getWeather",
			Description: "Gets weather info",
			InputSchema: map[string]any{"location": map[string]any{"type": "string"}},
		}},
		ToolChoice: &types.AnthropicToolChoice{Type: "auto"},
	}
	openAIPayload := anthropic.TranslateToOpenAI(anthropicPayload)
	if !isValidChatCompletionRequest(openAIPayload) {
		t.Fatalf("translated payload is invalid: %#v", openAIPayload)
	}
}

func TestTranslateInvalidTypesAnthropicPayload(t *testing.T) {
	anthropicPayload := types.AnthropicMessagesPayload{
		Model:    "gpt-4o",
		Messages: []types.AnthropicMessage{{Role: "user", Content: "Hello!"}},
		// intentionally leave temperature nil to simulate malformed input path
	}
	openAIPayload := anthropic.TranslateToOpenAI(anthropicPayload)
	if !isValidChatCompletionRequest(openAIPayload) {
		t.Fatalf("expected translated payload to remain structurally valid")
	}
}

func TestThinkingBlocksInAssistantMessages(t *testing.T) {
	anthropicPayload := types.AnthropicMessagesPayload{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []types.AnthropicMessage{
			{Role: "user", Content: "What is 2+2?"},
			{
				Role: "assistant",
				Content: []any{
					map[string]any{"type": "thinking", "thinking": "Let me think about this simple math problem..."},
					map[string]any{"type": "text", "text": "2+2 equals 4."},
				},
			},
		},
		MaxTokens: 100,
	}
	openAIPayload := anthropic.TranslateToOpenAI(anthropicPayload)
	if !isValidChatCompletionRequest(openAIPayload) {
		t.Fatalf("translated payload is invalid")
	}

	found := false
	for _, m := range openAIPayload.Messages {
		if m.Role == "assistant" {
			content, _ := m.Content.(string)
			if containsAll(content, []string{"Let me think about this simple math problem...", "2+2 equals 4."}) {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("assistant thinking content not found in translated payload")
	}
}

func TestThinkingBlocksWithToolCalls(t *testing.T) {
	anthropicPayload := types.AnthropicMessagesPayload{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []types.AnthropicMessage{
			{Role: "user", Content: "What's the weather?"},
			{
				Role: "assistant",
				Content: []any{
					map[string]any{"type": "thinking", "thinking": "I need to call the weather API to get current weather information."},
					map[string]any{"type": "text", "text": "I'll check the weather for you."},
					map[string]any{"type": "tool_use", "id": "call_123", "name": "get_weather", "input": map[string]any{"location": "New York"}},
				},
			},
		},
		MaxTokens: 100,
	}
	openAIPayload := anthropic.TranslateToOpenAI(anthropicPayload)
	if !isValidChatCompletionRequest(openAIPayload) {
		t.Fatalf("translated payload is invalid")
	}
	var assistant *types.Message
	for i := range openAIPayload.Messages {
		if openAIPayload.Messages[i].Role == "assistant" {
			assistant = &openAIPayload.Messages[i]
			break
		}
	}
	if assistant == nil {
		t.Fatalf("assistant message not found")
	}
	content, _ := assistant.Content.(string)
	if !containsAll(content, []string{"I need to call the weather API", "I'll check the weather for you."}) {
		t.Fatalf("assistant content missing expected text: %q", content)
	}
	if len(assistant.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(assistant.ToolCalls))
	}
	if assistant.ToolCalls[0].Function.Name != "get_weather" {
		t.Fatalf("unexpected tool call name: %s", assistant.ToolCalls[0].Function.Name)
	}
}
