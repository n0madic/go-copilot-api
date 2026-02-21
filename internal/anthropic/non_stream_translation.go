package anthropic

import (
	"encoding/json"
	"strings"

	"github.com/n0madic/go-copilot-api/internal/types"
	"github.com/n0madic/go-copilot-api/internal/util"
)

func TranslateToOpenAI(payload types.AnthropicMessagesPayload) types.ChatCompletionsPayload {
	result := types.ChatCompletionsPayload{
		Model:       translateModelName(payload.Model),
		Messages:    translateAnthropicMessagesToOpenAI(payload.Messages, payload.System),
		Temperature: payload.Temperature,
		TopP:        payload.TopP,
		Stream:      payload.Stream,
	}
	if payload.MaxTokens > 0 {
		v := payload.MaxTokens
		result.MaxTokens = &v
	}
	if len(payload.StopSequences) > 0 {
		result.Stop = payload.StopSequences
	}
	if payload.Metadata != nil && payload.Metadata.UserID != "" {
		user := payload.Metadata.UserID
		result.User = &user
	}
	if len(payload.Tools) > 0 {
		result.Tools = translateAnthropicToolsToOpenAI(payload.Tools)
	}
	if payload.ToolChoice != nil {
		result.ToolChoice = translateAnthropicToolChoiceToOpenAI(payload.ToolChoice)
	}
	return result
}

func translateModelName(model string) string {
	if strings.HasPrefix(model, "claude-sonnet-4-") {
		return normalizeClaudeFamilyModel(model, "claude-sonnet-4")
	}
	if strings.HasPrefix(model, "claude-opus-4-") {
		return normalizeClaudeFamilyModel(model, "claude-opus-4")
	}
	if strings.HasPrefix(model, "claude-haiku-4-") {
		return normalizeClaudeFamilyModel(model, "claude-haiku-4")
	}
	return model
}

func normalizeClaudeFamilyModel(model string, base string) string {
	prefix := base + "-"
	if !strings.HasPrefix(model, prefix) {
		return model
	}
	tail := strings.TrimPrefix(model, prefix)
	if tail == "" {
		return base
	}
	if util.IsDigits(tail) && len(tail) == 1 {
		return base + "." + tail
	}
	if util.IsDigits(tail) && len(tail) >= 6 {
		// date-like suffix from subagent requests
		return base
	}
	parts := strings.SplitN(tail, "-", 2)
	if len(parts) > 0 && util.IsDigits(parts[0]) && len(parts[0]) == 1 {
		return base + "." + parts[0]
	}
	return base
}

func translateAnthropicMessagesToOpenAI(messages []types.AnthropicMessage, system any) []types.Message {
	systemMessages := handleSystemPrompt(system)
	other := make([]types.Message, 0)
	for _, message := range messages {
		if message.Role == "user" {
			other = append(other, handleUserMessage(message)...)
		} else {
			other = append(other, handleAssistantMessage(message)...)
		}
	}
	return append(systemMessages, other...)
}

func handleSystemPrompt(system any) []types.Message {
	if system == nil {
		return nil
	}
	switch s := system.(type) {
	case string:
		return []types.Message{{Role: "system", Content: s}}
	case []any:
		chunks := make([]string, 0)
		for _, block := range s {
			bm, ok := block.(map[string]any)
			if !ok {
				continue
			}
			text, _ := bm["text"].(string)
			if text != "" {
				chunks = append(chunks, text)
			}
		}
		if len(chunks) == 0 {
			return nil
		}
		return []types.Message{{Role: "system", Content: strings.Join(chunks, "\n\n")}}
	default:
		return nil
	}
}

func handleUserMessage(message types.AnthropicMessage) []types.Message {
	newMessages := make([]types.Message, 0)
	blocks, ok := message.Content.([]any)
	if !ok {
		newMessages = append(newMessages, types.Message{Role: "user", Content: mapContent(message.Content)})
		return newMessages
	}

	toolResultBlocks := make([]map[string]any, 0)
	otherBlocks := make([]any, 0)
	for _, block := range blocks {
		bm, ok := block.(map[string]any)
		if !ok {
			continue
		}
		if bt, _ := bm["type"].(string); bt == "tool_result" {
			toolResultBlocks = append(toolResultBlocks, bm)
		} else {
			otherBlocks = append(otherBlocks, bm)
		}
	}

	for _, block := range toolResultBlocks {
		toolUseID, _ := block["tool_use_id"].(string)
		content := mapContent(block["content"])
		newMessages = append(newMessages, types.Message{Role: "tool", ToolCallID: toolUseID, Content: content})
	}
	if len(otherBlocks) > 0 {
		newMessages = append(newMessages, types.Message{Role: "user", Content: mapContent(otherBlocks)})
	}
	return newMessages
}

func handleAssistantMessage(message types.AnthropicMessage) []types.Message {
	blocks, ok := message.Content.([]any)
	if !ok {
		return []types.Message{{Role: "assistant", Content: mapContent(message.Content)}}
	}

	toolUse := make([]map[string]any, 0)
	textBlocks := make([]string, 0)
	thinkingBlocks := make([]string, 0)
	for _, b := range blocks {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		switch bm["type"] {
		case "tool_use":
			toolUse = append(toolUse, bm)
		case "text":
			if text, _ := bm["text"].(string); text != "" {
				textBlocks = append(textBlocks, text)
			}
		case "thinking":
			if thought, _ := bm["thinking"].(string); thought != "" {
				thinkingBlocks = append(thinkingBlocks, thought)
			}
		}
	}
	// thinking blocks come before text blocks to preserve reasoning context
	allText := strings.Join(append(thinkingBlocks, textBlocks...), "\n\n")
	if len(toolUse) > 0 {
		toolCalls := make([]types.ToolCall, 0, len(toolUse))
		for _, t := range toolUse {
			id, _ := t["id"].(string)
			name, _ := t["name"].(string)
			input := t["input"]
			b, _ := json.Marshal(input)
			toolCalls = append(toolCalls, types.ToolCall{ID: id, Type: "function", Function: types.ToolCallFn{Name: name, Arguments: string(b)}})
		}
		var content any
		if allText != "" {
			content = allText
		}
		return []types.Message{{Role: "assistant", Content: content, ToolCalls: toolCalls}}
	}
	return []types.Message{{Role: "assistant", Content: mapContent(message.Content)}}
}

func mapContent(content any) any {
	switch c := content.(type) {
	case string:
		return c
	case []any:
		hasImage := false
		for _, block := range c {
			bm, ok := block.(map[string]any)
			if !ok {
				continue
			}
			if bt, _ := bm["type"].(string); bt == "image" {
				hasImage = true
				break
			}
		}
		if !hasImage {
			parts := make([]string, 0)
			for _, block := range c {
				bm, ok := block.(map[string]any)
				if !ok {
					continue
				}
				switch bm["type"] {
				case "text":
					if text, _ := bm["text"].(string); text != "" {
						parts = append(parts, text)
					}
				case "thinking":
					if thought, _ := bm["thinking"].(string); thought != "" {
						parts = append(parts, thought)
					}
				}
			}
			return strings.Join(parts, "\n\n")
		}

		// Build []any directly instead of converting from intermediate []map[string]any.
		parts := make([]any, 0)
		for _, block := range c {
			bm, ok := block.(map[string]any)
			if !ok {
				continue
			}
			switch bm["type"] {
			case "text":
				text, _ := bm["text"].(string)
				parts = append(parts, map[string]any{"type": "text", "text": text})
			case "thinking":
				thought, _ := bm["thinking"].(string)
				parts = append(parts, map[string]any{"type": "text", "text": thought})
			case "image":
				source, _ := bm["source"].(map[string]any)
				mediaType, _ := source["media_type"].(string)
				data, _ := source["data"].(string)
				parts = append(parts, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": "data:" + mediaType + ";base64," + data,
					},
				})
			}
		}
		if len(parts) == 0 {
			return nil
		}
		return parts
	default:
		return nil
	}
}

func translateAnthropicToolsToOpenAI(tools []types.AnthropicTool) []types.Tool {
	out := make([]types.Tool, 0, len(tools))
	for _, t := range tools {
		out = append(out, types.Tool{
			Type: "function",
			Function: types.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return out
}

func translateAnthropicToolChoiceToOpenAI(choice *types.AnthropicToolChoice) any {
	if choice == nil {
		return nil
	}
	switch choice.Type {
	case "auto":
		return "auto"
	case "any":
		return "required"
	case "tool":
		if choice.Name != "" {
			return map[string]any{"type": "function", "function": map[string]any{"name": choice.Name}}
		}
		return nil
	case "none":
		return "none"
	default:
		return nil
	}
}

func TranslateToAnthropic(response types.ChatCompletionResponse) types.AnthropicResponse {
	allTextBlocks := make([]map[string]any, 0)
	allToolBlocks := make([]map[string]any, 0)
	stopReason := ""
	if len(response.Choices) > 0 {
		stopReason = response.Choices[0].FinishReason
	}

	for _, choice := range response.Choices {
		allTextBlocks = append(allTextBlocks, getAnthropicTextBlocks(choice.Message.Content)...)
		allToolBlocks = append(allToolBlocks, getAnthropicToolUseBlocks(choice.Message.ToolCalls)...)
		if choice.FinishReason == "tool_calls" || stopReason == "stop" {
			stopReason = choice.FinishReason
		}
	}

	usage := types.AnthropicResponseUsage{InputTokens: 0, OutputTokens: 0}
	if response.Usage != nil {
		cached := 0
		if response.Usage.PromptTokensDetails != nil {
			cached = response.Usage.PromptTokensDetails.CachedTokens
		}
		usage.InputTokens = response.Usage.PromptTokens - cached
		usage.OutputTokens = response.Usage.CompletionTokens
		if cached > 0 {
			usage.CacheReadInputTokens = &cached
		}
	}
	content := append(allTextBlocks, allToolBlocks...)

	return types.AnthropicResponse{
		ID:           response.ID,
		Type:         "message",
		Role:         "assistant",
		Model:        response.Model,
		Content:      content,
		StopReason:   MapOpenAIStopReasonToAnthropic(stopReason),
		StopSequence: nil,
		Usage:        usage,
	}
}

func getAnthropicTextBlocks(content any) []map[string]any {
	switch c := content.(type) {
	case string:
		if c == "" {
			return nil
		}
		return []map[string]any{{"type": "text", "text": c}}
	case []any:
		out := make([]map[string]any, 0)
		for _, part := range c {
			pm, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if pm["type"] == "text" {
				text, _ := pm["text"].(string)
				out = append(out, map[string]any{"type": "text", "text": text})
			}
		}
		return out
	default:
		return nil
	}
}

func getAnthropicToolUseBlocks(toolCalls []types.ToolCall) []map[string]any {
	if len(toolCalls) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(toolCalls))
	for _, call := range toolCalls {
		input := map[string]any{}
		_ = json.Unmarshal([]byte(call.Function.Arguments), &input)
		out = append(out, map[string]any{
			"type":  "tool_use",
			"id":    call.ID,
			"name":  call.Function.Name,
			"input": input,
		})
	}
	return out
}
