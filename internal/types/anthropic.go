package types

type AnthropicMessagesPayload struct {
	Model         string               `json:"model"`
	Messages      []AnthropicMessage   `json:"messages"`
	MaxTokens     int                  `json:"max_tokens"`
	System        any                  `json:"system,omitempty"`
	Metadata      *AnthropicMetadata   `json:"metadata,omitempty"`
	StopSequences []string             `json:"stop_sequences,omitempty"`
	Stream        *bool                `json:"stream,omitempty"`
	Temperature   *float64             `json:"temperature,omitempty"`
	TopP          *float64             `json:"top_p,omitempty"`
	TopK          *int                 `json:"top_k,omitempty"`
	Tools         []AnthropicTool      `json:"tools,omitempty"`
	ToolChoice    *AnthropicToolChoice `json:"tool_choice,omitempty"`
	Thinking      map[string]any       `json:"thinking,omitempty"`
	ServiceTier   string               `json:"service_tier,omitempty"`
}

type AnthropicMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

type AnthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type AnthropicTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type AnthropicImageBlock struct {
	Type   string               `json:"type"`
	Source AnthropicImageSource `json:"source"`
}

type AnthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type AnthropicToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   *bool  `json:"is_error,omitempty"`
}

type AnthropicToolUseBlock struct {
	Type  string         `json:"type"`
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

type AnthropicThinkingBlock struct {
	Type     string `json:"type"`
	Thinking string `json:"thinking"`
}

type AnthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

type AnthropicResponse struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Role         string                 `json:"role"`
	Content      []map[string]any       `json:"content"`
	Model        string                 `json:"model"`
	StopReason   any                    `json:"stop_reason"`
	StopSequence any                    `json:"stop_sequence"`
	Usage        AnthropicResponseUsage `json:"usage"`
}

type AnthropicResponseUsage struct {
	InputTokens              int  `json:"input_tokens"`
	OutputTokens             int  `json:"output_tokens"`
	CacheCreationInputTokens *int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     *int `json:"cache_read_input_tokens,omitempty"`
}

type AnthropicStreamState struct {
	MessageStartSent  bool
	ContentBlockIndex int
	ContentBlockOpen  bool
	ToolCalls         map[int]ToolCallState
}

type ToolCallState struct {
	ID                  string
	Name                string
	AnthropicBlockIndex int
}

// stream event payloads

type AnthropicMessageStartEvent struct {
	Type    string         `json:"type"`
	Message map[string]any `json:"message"`
}

type AnthropicContentBlockStartEvent struct {
	Type         string         `json:"type"`
	Index        int            `json:"index"`
	ContentBlock map[string]any `json:"content_block"`
}

type AnthropicContentBlockDeltaEvent struct {
	Type  string         `json:"type"`
	Index int            `json:"index"`
	Delta map[string]any `json:"delta"`
}

type AnthropicContentBlockStopEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type AnthropicMessageDeltaEvent struct {
	Type  string         `json:"type"`
	Delta map[string]any `json:"delta"`
	Usage map[string]any `json:"usage,omitempty"`
}

type AnthropicMessageStopEvent struct {
	Type string `json:"type"`
}

type AnthropicErrorEvent struct {
	Type  string         `json:"type"`
	Error map[string]any `json:"error"`
}
