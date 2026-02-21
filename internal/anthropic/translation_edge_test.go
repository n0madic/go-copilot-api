package anthropic_test

import (
	"testing"

	"github.com/n0madic/go-copilot-api/internal/anthropic"
	"github.com/n0madic/go-copilot-api/internal/types"
)

func TestModelNormalizationForSubagentModels(t *testing.T) {
	payload := types.AnthropicMessagesPayload{
		Model: "claude-sonnet-4-20250101",
		Messages: []types.AnthropicMessage{
			{Role: "user", Content: "hello"},
		},
		MaxTokens: 10,
	}
	translated := anthropic.TranslateToOpenAI(payload)
	if translated.Model != "claude-sonnet-4" {
		t.Fatalf("expected claude-sonnet-4, got %q", translated.Model)
	}

	payload.Model = "claude-sonnet-4-6"
	translated = anthropic.TranslateToOpenAI(payload)
	if translated.Model != "claude-sonnet-4.6" {
		t.Fatalf("expected claude-sonnet-4.6, got %q", translated.Model)
	}

	payload.Model = "claude-opus-4-20250101"
	translated = anthropic.TranslateToOpenAI(payload)
	if translated.Model != "claude-opus-4" {
		t.Fatalf("expected claude-opus-4, got %q", translated.Model)
	}

	payload.Model = "claude-opus-4-6"
	translated = anthropic.TranslateToOpenAI(payload)
	if translated.Model != "claude-opus-4.6" {
		t.Fatalf("expected claude-opus-4.6, got %q", translated.Model)
	}
}

func TestToolResultComesBeforeUserMessage(t *testing.T) {
	payload := types.AnthropicMessagesPayload{
		Model: "gpt-4o",
		Messages: []types.AnthropicMessage{
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": "tool_result", "tool_use_id": "tool_1", "content": "ok"},
					map[string]any{"type": "text", "text": "continue"},
				},
			},
		},
		MaxTokens: 10,
	}
	translated := anthropic.TranslateToOpenAI(payload)
	if len(translated.Messages) < 2 {
		t.Fatalf("expected at least two messages, got %d", len(translated.Messages))
	}
	if translated.Messages[0].Role != "tool" {
		t.Fatalf("expected first message to be tool, got %s", translated.Messages[0].Role)
	}
	if translated.Messages[1].Role != "user" {
		t.Fatalf("expected second message to be user, got %s", translated.Messages[1].Role)
	}
}

func TestAssistantToolUseWithoutTextKeepsEmptyStringContent(t *testing.T) {
	payload := types.AnthropicMessagesPayload{
		Model: "gpt-4o",
		Messages: []types.AnthropicMessage{
			{
				Role: "assistant",
				Content: []any{
					map[string]any{
						"type":  "tool_use",
						"id":    "call_1",
						"name":  "get_weather",
						"input": map[string]any{"city": "Berlin"},
					},
				},
			},
		},
		MaxTokens: 10,
	}

	translated := anthropic.TranslateToOpenAI(payload)
	if len(translated.Messages) != 1 {
		t.Fatalf("expected one translated message, got %d", len(translated.Messages))
	}
	msg := translated.Messages[0]
	if msg.Role != "assistant" {
		t.Fatalf("expected assistant role, got %q", msg.Role)
	}
	if msg.Content != "" {
		t.Fatalf("expected empty string content instead of nil, got %#v", msg.Content)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %d", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].ID == "" {
		t.Fatalf("expected non-empty tool call id")
	}
}
