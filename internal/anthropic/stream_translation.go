package anthropic

import "github.com/n0madic/go-copilot-api/internal/types"

func isToolBlockOpen(state *types.AnthropicStreamState) bool {
	if !state.ContentBlockOpen {
		return false
	}
	for _, tc := range state.ToolCalls {
		if tc.AnthropicBlockIndex == state.ContentBlockIndex {
			return true
		}
	}
	return false
}

func TranslateChunkToAnthropicEvents(chunk types.ChatCompletionChunk, state *types.AnthropicStreamState) []map[string]any {
	events := make([]map[string]any, 0)
	if len(chunk.Choices) == 0 {
		return events
	}
	choice := chunk.Choices[0]
	delta := choice.Delta

	if !state.MessageStartSent {
		inputTokens := 0
		cacheRead := 0
		if chunk.Usage != nil {
			cached := 0
			if chunk.Usage.PromptTokensDetails != nil {
				cached = chunk.Usage.PromptTokensDetails.CachedTokens
			}
			inputTokens = chunk.Usage.PromptTokens - cached
			cacheRead = cached
		}
		message := map[string]any{
			"id":            chunk.ID,
			"type":          "message",
			"role":          "assistant",
			"content":       []any{},
			"model":         chunk.Model,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  inputTokens,
				"output_tokens": 0,
			},
		}
		if cacheRead > 0 {
			message["usage"].(map[string]any)["cache_read_input_tokens"] = cacheRead
		}
		events = append(events, map[string]any{"type": "message_start", "message": message})
		state.MessageStartSent = true
	}

	if delta.Content != nil && *delta.Content != "" {
		if isToolBlockOpen(state) {
			events = append(events, map[string]any{"type": "content_block_stop", "index": state.ContentBlockIndex})
			state.ContentBlockIndex++
			state.ContentBlockOpen = false
		}
		if !state.ContentBlockOpen {
			events = append(events, map[string]any{
				"type":  "content_block_start",
				"index": state.ContentBlockIndex,
				"content_block": map[string]any{
					"type": "text",
					"text": "",
				},
			})
			state.ContentBlockOpen = true
		}
		events = append(events, map[string]any{
			"type":  "content_block_delta",
			"index": state.ContentBlockIndex,
			"delta": map[string]any{
				"type": "text_delta",
				"text": *delta.Content,
			},
		})
	}

	if len(delta.ToolCalls) > 0 {
		for _, toolCall := range delta.ToolCalls {
			if toolCall.ID != "" && toolCall.Function != nil && toolCall.Function.Name != "" {
				if state.ContentBlockOpen {
					events = append(events, map[string]any{"type": "content_block_stop", "index": state.ContentBlockIndex})
					state.ContentBlockIndex++
					state.ContentBlockOpen = false
				}
				anthropicBlockIndex := state.ContentBlockIndex
				if state.ToolCalls == nil {
					state.ToolCalls = map[int]types.ToolCallState{}
				}
				state.ToolCalls[toolCall.Index] = types.ToolCallState{
					ID:                  toolCall.ID,
					Name:                toolCall.Function.Name,
					AnthropicBlockIndex: anthropicBlockIndex,
				}
				events = append(events, map[string]any{
					"type":  "content_block_start",
					"index": anthropicBlockIndex,
					"content_block": map[string]any{
						"type":  "tool_use",
						"id":    toolCall.ID,
						"name":  toolCall.Function.Name,
						"input": map[string]any{},
					},
				})
				state.ContentBlockOpen = true
			}
			if toolCall.Function != nil && toolCall.Function.Arguments != "" {
				if info, ok := state.ToolCalls[toolCall.Index]; ok {
					events = append(events, map[string]any{
						"type":  "content_block_delta",
						"index": info.AnthropicBlockIndex,
						"delta": map[string]any{
							"type":         "input_json_delta",
							"partial_json": toolCall.Function.Arguments,
						},
					})
				}
			}
		}
	}

	if choice.FinishReason != nil {
		if state.ContentBlockOpen {
			events = append(events, map[string]any{"type": "content_block_stop", "index": state.ContentBlockIndex})
			state.ContentBlockOpen = false
		}
		inputTokens := 0
		outputTokens := 0
		cacheRead := 0
		if chunk.Usage != nil {
			cached := 0
			if chunk.Usage.PromptTokensDetails != nil {
				cached = chunk.Usage.PromptTokensDetails.CachedTokens
			}
			inputTokens = chunk.Usage.PromptTokens - cached
			outputTokens = chunk.Usage.CompletionTokens
			cacheRead = cached
		}
		usage := map[string]any{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		}
		if cacheRead > 0 {
			usage["cache_read_input_tokens"] = cacheRead
		}
		events = append(events,
			map[string]any{
				"type": "message_delta",
				"delta": map[string]any{
					"stop_reason":   MapOpenAIStopReasonToAnthropic(*choice.FinishReason),
					"stop_sequence": nil,
				},
				"usage": usage,
			},
			map[string]any{"type": "message_stop"},
		)
	}

	return events
}

func TranslateErrorToAnthropicErrorEvent() map[string]any {
	return map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    "api_error",
			"message": "An unexpected error occurred during streaming.",
		},
	}
}
