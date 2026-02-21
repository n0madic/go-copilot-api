package responses

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/n0madic/go-copilot-api/internal/types"
)

func BuildChatCompletionsPayload(req types.ResponsesRequest, history []types.Message) (types.ChatCompletionsPayload, error) {
	messages := make([]types.Message, 0, len(history)+8)
	messages = append(messages, cloneMessages(history)...)

	if req.Instructions != nil && strings.TrimSpace(*req.Instructions) != "" {
		messages = append(messages, types.Message{Role: "system", Content: *req.Instructions})
	}

	inputMessages, err := inputToMessages(req.Input)
	if err != nil {
		return types.ChatCompletionsPayload{}, err
	}
	messages = append(messages, inputMessages...)

	if len(messages) == 0 {
		return types.ChatCompletionsPayload{}, errors.New("request must include input or previous_response_id")
	}

	payload := types.ChatCompletionsPayload{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
		ToolChoice:  normalizeToolChoice(req.ToolChoice),
		User:        req.User,
	}

	if req.MaxOutputTokens != nil {
		v := *req.MaxOutputTokens
		payload.MaxTokens = &v
	}
	if len(req.Tools) > 0 {
		payload.Tools = translateTools(req.Tools)
	}
	if err := validateToolMessageSequence(payload.Messages); err != nil {
		return types.ChatCompletionsPayload{}, err
	}

	return payload, nil
}

func CompletionToResponse(req types.ResponsesRequest, completion types.ChatCompletionResponse, responseID string) (types.ResponsesResponse, types.Message) {
	createdAt := completion.Created
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}

	assistant := normalizeAssistantToolCalls(assistantMessageFromCompletion(completion))
	output, outputText := outputFromAssistant(assistant)

	model := completion.Model
	if model == "" {
		model = req.Model
	}

	resp := types.ResponsesResponse{
		ID:                 responseID,
		Object:             "response",
		CreatedAt:          createdAt,
		Status:             "completed",
		Model:              model,
		Output:             output,
		OutputText:         outputText,
		Usage:              mapUsage(completion.Usage),
		PreviousResponseID: req.PreviousResponseID,
		Error:              nil,
		IncompleteDetails:  nil,
	}
	return resp, assistant
}

func assistantMessageFromCompletion(completion types.ChatCompletionResponse) types.Message {
	if len(completion.Choices) == 0 {
		return types.Message{Role: "assistant", Content: ""}
	}
	msg := completion.Choices[0].Message
	if msg.Role == "" {
		msg.Role = "assistant"
	}
	content := msg.Content
	if content == nil {
		content = ""
	}
	return types.Message{
		Role:      msg.Role,
		Content:   content,
		ToolCalls: msg.ToolCalls,
	}
}

func normalizeAssistantToolCalls(msg types.Message) types.Message {
	if len(msg.ToolCalls) == 0 {
		return msg
	}
	toolCalls := make([]types.ToolCall, len(msg.ToolCalls))
	copy(toolCalls, msg.ToolCalls)
	for i := range toolCalls {
		if strings.TrimSpace(toolCalls[i].ID) == "" {
			toolCalls[i].ID = newCallID()
		}
		if strings.TrimSpace(toolCalls[i].Type) == "" {
			toolCalls[i].Type = "function"
		}
	}
	msg.ToolCalls = toolCalls
	return msg
}

func outputFromAssistant(msg types.Message) ([]types.ResponsesOutputItem, string) {
	output := make([]types.ResponsesOutputItem, 0, 1+len(msg.ToolCalls))
	outputText := extractAssistantText(msg.Content)

	if outputText != "" || len(msg.ToolCalls) == 0 {
		output = append(output, types.ResponsesOutputItem{
			ID:     newMessageID(),
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			Content: []types.ResponsesContentPart{{
				Type:        "output_text",
				Text:        outputText,
				Annotations: []any{},
			}},
		})
	}

	for _, tc := range msg.ToolCalls {
		output = append(output, types.ResponsesOutputItem{
			ID:        newFunctionID(),
			Type:      "function_call",
			Status:    "completed",
			CallID:    tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return output, outputText
}

func extractAssistantText(content any) string {
	switch v := content.(type) {
	case nil:
		return ""
	case string:
		return v
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			typeValue, _ := m["type"].(string)
			switch typeValue {
			case "text", "output_text", "input_text":
				if text, _ := m["text"].(string); text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n\n")
	default:
		buf, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(buf)
	}
}

func mapUsage(usage *types.Usage) *types.ResponsesUsage {
	if usage == nil {
		return nil
	}

	var inputDetails any
	if usage.PromptTokensDetails != nil {
		inputDetails = map[string]any{"cached_tokens": usage.PromptTokensDetails.CachedTokens}
	}

	return &types.ResponsesUsage{
		InputTokens:         usage.PromptTokens,
		OutputTokens:        usage.CompletionTokens,
		TotalTokens:         usage.TotalTokens,
		InputTokensDetails:  inputDetails,
		OutputTokensDetails: usage.CompletionTokensDetails,
	}
}

func inputToMessages(input any) ([]types.Message, error) {
	if input == nil {
		return nil, nil
	}

	switch v := input.(type) {
	case string:
		return []types.Message{{Role: "user", Content: v}}, nil
	case []any:
		return itemsToMessages(v)
	case map[string]any:
		return itemToMessages(v)
	default:
		return nil, fmt.Errorf("unsupported input type %T", input)
	}
}

func itemsToMessages(items []any) ([]types.Message, error) {
	messages := make([]types.Message, 0, len(items))
	for _, item := range items {
		msgBatch, err := itemToMessages(item)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msgBatch...)
	}
	return messages, nil
}

func itemToMessages(item any) ([]types.Message, error) {
	m, ok := item.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("input item must be object, got %T", item)
	}

	typeValue, _ := m["type"].(string)
	if typeValue == "function_call_output" {
		callID, _ := m["call_id"].(string)
		if callID == "" {
			return nil, errors.New("function_call_output requires call_id")
		}
		return []types.Message{{
			Role:       "tool",
			ToolCallID: callID,
			Content:    stringifyValue(m["output"]),
		}}, nil
	}

	if role, ok := m["role"].(string); ok && role != "" {
		content, err := normalizeMessageContent(m["content"])
		if err != nil {
			return nil, err
		}
		return []types.Message{{Role: role, Content: content}}, nil
	}

	if typeValue == "message" {
		role, _ := m["role"].(string)
		if role == "" {
			role = "user"
		}
		content, err := normalizeMessageContent(m["content"])
		if err != nil {
			return nil, err
		}
		return []types.Message{{Role: role, Content: content}}, nil
	}

	return nil, fmt.Errorf("unsupported input item type %q", typeValue)
}

func normalizeMessageContent(content any) (any, error) {
	switch v := content.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case []any:
		parts := make([]any, 0, len(v))
		textOnly := true
		textParts := make([]string, 0, len(v))

		for _, rawPart := range v {
			part, ok := rawPart.(map[string]any)
			if !ok {
				continue
			}
			partType, _ := part["type"].(string)
			switch partType {
			case "input_text", "output_text", "text":
				text, _ := part["text"].(string)
				if text == "" {
					continue
				}
				textParts = append(textParts, text)
				parts = append(parts, map[string]any{"type": "text", "text": text})
			case "input_image", "image_url":
				url, ok := extractImageURL(part)
				if !ok {
					return nil, errors.New("image part must include image_url")
				}
				textOnly = false
				parts = append(parts, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": url,
					},
				})
			}
		}

		if textOnly {
			return strings.Join(textParts, "\n\n"), nil
		}
		return parts, nil
	case map[string]any:
		partType, _ := v["type"].(string)
		switch partType {
		case "input_text", "output_text", "text":
			text, _ := v["text"].(string)
			return text, nil
		case "input_image", "image_url":
			url, ok := extractImageURL(v)
			if !ok {
				return nil, errors.New("image part must include image_url")
			}
			return []any{map[string]any{"type": "image_url", "image_url": map[string]any{"url": url}}}, nil
		default:
			return stringifyValue(v), nil
		}
	default:
		return stringifyValue(v), nil
	}
}

func extractImageURL(part map[string]any) (string, bool) {
	if url, ok := part["image_url"].(string); ok && url != "" {
		return url, true
	}
	if imageURLMap, ok := part["image_url"].(map[string]any); ok {
		if url, ok := imageURLMap["url"].(string); ok && url != "" {
			return url, true
		}
	}
	return "", false
}

func stringifyValue(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	default:
		buf, err := json.Marshal(value)
		if err != nil {
			return ""
		}
		return string(buf)
	}
}

func translateTools(tools []types.ResponsesTool) []types.Tool {
	result := make([]types.Tool, 0, len(tools))
	for _, tool := range tools {
		if tool.Type != "" && tool.Type != "function" {
			continue
		}

		if tool.Function != nil {
			result = append(result, types.Tool{Type: "function", Function: *tool.Function})
			continue
		}

		if strings.TrimSpace(tool.Name) == "" {
			continue
		}

		parameters := tool.Parameters
		if parameters == nil {
			parameters = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		result = append(result, types.Tool{
			Type: "function",
			Function: types.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  parameters,
			},
		})
	}
	return result
}

func normalizeToolChoice(choice any) any {
	m, ok := choice.(map[string]any)
	if !ok {
		return choice
	}
	typeValue, _ := m["type"].(string)
	if typeValue != "function" {
		return choice
	}
	if _, hasFunction := m["function"]; hasFunction {
		return choice
	}
	name, _ := m["name"].(string)
	if name == "" {
		return choice
	}
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name": name,
		},
	}
}

func validateToolMessageSequence(messages []types.Message) error {
	pending := make(map[string]struct{})
	for i, msg := range messages {
		switch msg.Role {
		case "assistant":
			for _, tc := range msg.ToolCalls {
				id := strings.TrimSpace(tc.ID)
				if id == "" {
					continue
				}
				pending[id] = struct{}{}
			}
		case "tool":
			toolCallID := strings.TrimSpace(msg.ToolCallID)
			if toolCallID == "" {
				return fmt.Errorf("tool message at index %d requires tool_call_id", i)
			}
			if _, ok := pending[toolCallID]; !ok {
				return fmt.Errorf("tool message at index %d does not match any previous assistant tool_call", i)
			}
			delete(pending, toolCallID)
		}
	}
	return nil
}
