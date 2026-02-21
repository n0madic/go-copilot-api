package tokenizer

import (
	"encoding/json"
	"log/slog"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/n0madic/go-copilot-api/internal/types"
)

type TokenCount struct {
	Input  int
	Output int
}

type modelConstants struct {
	funcInit int
	propInit int
	propKey  int
	enumInit int
	enumItem int
	funcEnd  int
}

func GetTokenizerFromModel(model types.Model) string {
	if model.Capabilities.Tokenizer != "" {
		return model.Capabilities.Tokenizer
	}
	return "o200k_base"
}

func getModelConstants(model types.Model) modelConstants {
	if model.ID == "gpt-3.5-turbo" || model.ID == "gpt-4" {
		return modelConstants{funcInit: 10, propInit: 3, propKey: 3, enumInit: -3, enumItem: 3, funcEnd: 12}
	}
	return modelConstants{funcInit: 7, propInit: 3, propKey: 3, enumInit: -3, enumItem: 3, funcEnd: 12}
}

func tokenLen(s string) int {
	if s == "" {
		return 0
	}
	runes := utf8.RuneCountInString(s)
	if runes <= 4 {
		return 1
	}
	// Balanced approximation: better than raw rune or bytes for mixed content.
	return int(math.Ceil(float64(runes) / 4.0))
}

func calculateToolCallsTokens(toolCalls []types.ToolCall, constants modelConstants) int {
	tokens := 0
	for _, call := range toolCalls {
		tokens += constants.funcInit
		b, err := json.Marshal(call)
		if err != nil {
			slog.Warn("failed to marshal tool call for token counting", "error", err)
			continue
		}
		tokens += tokenLen(string(b))
	}
	tokens += constants.funcEnd
	return tokens
}

func calculateContentPartsTokens(contentParts []any) int {
	tokens := 0
	for _, part := range contentParts {
		pm, ok := part.(map[string]any)
		if !ok {
			continue
		}
		partType, _ := pm["type"].(string)
		if partType == "image_url" {
			imageURL, _ := pm["image_url"].(map[string]any)
			url, _ := imageURL["url"].(string)
			tokens += tokenLen(url) + 85
			continue
		}
		text, _ := pm["text"].(string)
		tokens += tokenLen(text)
	}
	return tokens
}

func calculateMessageTokens(message types.Message, constants modelConstants) int {
	tokens := 3
	if message.Role != "" {
		tokens += tokenLen(message.Role)
	}
	if message.Name != "" {
		tokens += tokenLen(message.Name)
		tokens += 1
	}
	if message.ToolCallID != "" {
		tokens += tokenLen(message.ToolCallID)
	}
	if len(message.ToolCalls) > 0 {
		tokens += calculateToolCallsTokens(message.ToolCalls, constants)
	}
	switch content := message.Content.(type) {
	case string:
		tokens += tokenLen(content)
	case []any:
		tokens += calculateContentPartsTokens(content)
	case nil:
		// no-op
	default:
		b, err := json.Marshal(content)
		if err != nil {
			slog.Warn("failed to marshal message content for token counting", "error", err)
			break
		}
		tokens += tokenLen(string(b))
	}
	return tokens
}

func calculateTokens(messages []types.Message, constants modelConstants) int {
	if len(messages) == 0 {
		return 0
	}
	numTokens := 0
	for _, msg := range messages {
		numTokens += calculateMessageTokens(msg, constants)
	}
	numTokens += 3
	return numTokens
}

func calculateParameterTokens(key string, prop any, constants modelConstants) int {
	tokens := constants.propKey
	param, ok := prop.(map[string]any)
	if !ok {
		return tokens
	}
	paramType, _ := param["type"].(string)
	if paramType == "" {
		paramType = "string"
	}
	paramDesc, _ := param["description"].(string)
	paramDesc = strings.TrimSuffix(paramDesc, ".")
	if enumValues, ok := param["enum"].([]any); ok {
		tokens += constants.enumInit
		for _, item := range enumValues {
			tokens += constants.enumItem
			tokens += tokenLen(toString(item))
		}
	}
	tokens += tokenLen(key + ":" + paramType + ":" + paramDesc)
	for propName, propValue := range param {
		if propName == "type" || propName == "description" || propName == "enum" {
			continue
		}
		tokens += tokenLen(propName + ":" + toString(propValue))
	}
	return tokens
}

func calculateParametersTokens(parameters map[string]any, constants modelConstants) int {
	tokens := 0
	for key, value := range parameters {
		if key == "properties" {
			properties, ok := value.(map[string]any)
			if !ok || len(properties) == 0 {
				continue
			}
			tokens += constants.propInit
			for propKey, propValue := range properties {
				tokens += calculateParameterTokens(propKey, propValue, constants)
			}
			continue
		}
		tokens += tokenLen(key + ":" + toString(value))
	}
	return tokens
}

func calculateToolTokens(tool types.Tool, constants modelConstants) int {
	tokens := constants.funcInit
	fn := tool.Function
	desc := strings.TrimSuffix(fn.Description, ".")
	tokens += tokenLen(fn.Name + ":" + desc)
	tokens += calculateParametersTokens(fn.Parameters, constants)
	return tokens
}

func NumTokensForTools(tools []types.Tool, constants modelConstants) int {
	count := 0
	for _, tool := range tools {
		count += calculateToolTokens(tool, constants)
	}
	count += constants.funcEnd
	return count
}

func GetTokenCount(payload types.ChatCompletionsPayload, model types.Model) TokenCount {
	constants := getModelConstants(model)
	inputMessages := make([]types.Message, 0, len(payload.Messages))
	outputMessages := make([]types.Message, 0, len(payload.Messages))
	for _, msg := range payload.Messages {
		if msg.Role == "assistant" {
			outputMessages = append(outputMessages, msg)
		} else {
			inputMessages = append(inputMessages, msg)
		}
	}
	input := calculateTokens(inputMessages, constants)
	if len(payload.Tools) > 0 {
		input += NumTokensForTools(payload.Tools, constants)
	}
	output := calculateTokens(outputMessages, constants)
	return TokenCount{Input: input, Output: output}
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
