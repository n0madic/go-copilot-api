package responses

import (
	"testing"

	"github.com/n0madic/go-copilot-api/internal/types"
)

func TestStreamTranslatorTextFlow(t *testing.T) {
	translator := NewStreamTranslator("resp_1", "gpt-5-mini", nil, 123)

	created := translator.CreatedEvent()
	if created.Name != "response.created" {
		t.Fatalf("unexpected created event: %q", created.Name)
	}

	chunk1 := types.ChatCompletionChunk{
		Model: "gpt-5-mini",
		Choices: []types.ChoiceChunk{{
			Index: 0,
			Delta: types.Delta{Content: strPtr("Hel")},
		}},
	}
	chunk2 := types.ChatCompletionChunk{
		Model: "gpt-5-mini",
		Choices: []types.ChoiceChunk{{
			Index:        0,
			Delta:        types.Delta{Content: strPtr("lo")},
			FinishReason: strPtr("stop"),
		}},
		Usage: &types.Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5},
	}

	events := make([]StreamEvent, 0)
	events = append(events, translator.HandleChunk(chunk1)...)
	events = append(events, translator.HandleChunk(chunk2)...)

	names := eventNames(events)
	mustContainEvent(t, names, "response.output_item.added")
	mustContainEvent(t, names, "response.content_part.added")
	mustContainEvent(t, names, "response.output_text.delta")
	mustContainEvent(t, names, "response.output_text.done")
	mustContainEvent(t, names, "response.output_item.done")
	mustContainEvent(t, names, "response.completed")

	if !translator.Completed() {
		t.Fatalf("expected translator to be completed")
	}
	assistant := translator.AssistantMessage()
	if assistant.Role != "assistant" {
		t.Fatalf("unexpected assistant role: %q", assistant.Role)
	}
	if assistant.Content != "Hello" {
		t.Fatalf("unexpected assistant content: %#v", assistant.Content)
	}
}

func TestStreamTranslatorFunctionCallFlow(t *testing.T) {
	translator := NewStreamTranslator("resp_2", "gpt-5-mini", nil, 123)

	chunk1 := types.ChatCompletionChunk{
		Choices: []types.ChoiceChunk{{
			Index: 0,
			Delta: types.Delta{ToolCalls: []types.ToolCallDelta{{
				Index: 0,
				ID:    "call_1",
				Function: &types.ToolCallFnDelta{
					Name: "get_weather",
				},
			}}},
		}},
	}
	chunk2 := types.ChatCompletionChunk{
		Choices: []types.ChoiceChunk{{
			Index: 0,
			Delta: types.Delta{ToolCalls: []types.ToolCallDelta{{
				Index: 0,
				Function: &types.ToolCallFnDelta{
					Arguments: `{"city":"Berlin"}`,
				},
			}}},
		}},
	}
	chunk3 := types.ChatCompletionChunk{
		Choices: []types.ChoiceChunk{{
			Index:        0,
			Delta:        types.Delta{},
			FinishReason: strPtr("tool_calls"),
		}},
	}

	events := make([]StreamEvent, 0)
	events = append(events, translator.HandleChunk(chunk1)...)
	events = append(events, translator.HandleChunk(chunk2)...)
	events = append(events, translator.HandleChunk(chunk3)...)

	names := eventNames(events)
	mustContainEvent(t, names, "response.output_item.added")
	mustContainEvent(t, names, "response.function_call_arguments.delta")
	mustContainEvent(t, names, "response.function_call_arguments.done")
	mustContainEvent(t, names, "response.output_item.done")
	mustContainEvent(t, names, "response.completed")

	assistant := translator.AssistantMessage()
	if len(assistant.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %d", len(assistant.ToolCalls))
	}
	if assistant.ToolCalls[0].Function.Arguments != `{"city":"Berlin"}` {
		t.Fatalf("unexpected tool arguments: %q", assistant.ToolCalls[0].Function.Arguments)
	}
}

func eventNames(events []StreamEvent) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		out = append(out, event.Name)
	}
	return out
}

func mustContainEvent(t *testing.T, names []string, want string) {
	t.Helper()
	for _, name := range names {
		if name == want {
			return
		}
	}
	t.Fatalf("expected event %q in %v", want, names)
}

func strPtr(v string) *string {
	return &v
}
