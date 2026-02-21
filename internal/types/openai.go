package types

import "encoding/json"

type ChatCompletionsPayload struct {
	Messages         []Message                  `json:"messages"`
	Model            string                     `json:"model"`
	Temperature      *float64                   `json:"temperature,omitempty"`
	TopP             *float64                   `json:"top_p,omitempty"`
	MaxTokens        *int                       `json:"max_tokens,omitempty"`
	Stop             any                        `json:"stop,omitempty"`
	N                *int                       `json:"n,omitempty"`
	Stream           *bool                      `json:"stream,omitempty"`
	FrequencyPenalty *float64                   `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64                   `json:"presence_penalty,omitempty"`
	LogitBias        map[string]float64         `json:"logit_bias,omitempty"`
	Logprobs         *bool                      `json:"logprobs,omitempty"`
	ResponseFormat   *ResponseFormat            `json:"response_format,omitempty"`
	Seed             *int                       `json:"seed,omitempty"`
	Tools            []Tool                     `json:"tools,omitempty"`
	ToolChoice       any                        `json:"tool_choice,omitempty"`
	User             *string                    `json:"user,omitempty"`
	Extra            map[string]json.RawMessage `json:"-"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string     `json:"id"`
	Type     string     `json:"type"`
	Function ToolCallFn `json:"function"`
}

type ToolCallFn struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ContentPart struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	ImageURL map[string]any `json:"image_url,omitempty"`
}

type ChatCompletionResponse struct {
	ID                string               `json:"id"`
	Object            string               `json:"object"`
	Created           int64                `json:"created"`
	Model             string               `json:"model"`
	Choices           []ChoiceNonStreaming `json:"choices"`
	SystemFingerprint string               `json:"system_fingerprint,omitempty"`
	Usage             *Usage               `json:"usage,omitempty"`
}

type ChoiceNonStreaming struct {
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message"`
	Logprobs     any             `json:"logprobs"`
	FinishReason string          `json:"finish_reason"`
}

type ResponseMessage struct {
	Role      string     `json:"role"`
	Content   any        `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ChatCompletionChunk struct {
	ID                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	Choices           []ChoiceChunk `json:"choices"`
	SystemFingerprint string        `json:"system_fingerprint,omitempty"`
	Usage             *Usage        `json:"usage,omitempty"`
}

type ChoiceChunk struct {
	Index        int     `json:"index"`
	Delta        Delta   `json:"delta"`
	FinishReason *string `json:"finish_reason"`
	Logprobs     any     `json:"logprobs"`
}

type Delta struct {
	Content   *string         `json:"content,omitempty"`
	Role      string          `json:"role,omitempty"`
	ToolCalls []ToolCallDelta `json:"tool_calls,omitempty"`
}

type ToolCallDelta struct {
	Index    int              `json:"index"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function *ToolCallFnDelta `json:"function,omitempty"`
}

type ToolCallFnDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type Usage struct {
	PromptTokens            int                  `json:"prompt_tokens"`
	CompletionTokens        int                  `json:"completion_tokens"`
	TotalTokens             int                  `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails map[string]int       `json:"completion_tokens_details,omitempty"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type EmbeddingRequest struct {
	Input any    `json:"input"`
	Model string `json:"model"`
}

type Embedding struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type EmbeddingResponse struct {
	Object string      `json:"object"`
	Data   []Embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}
