package types

type ResponsesRequest struct {
	Model              string          `json:"model"`
	Input              any             `json:"input,omitempty"`
	Instructions       *string         `json:"instructions,omitempty"`
	MaxOutputTokens    *int            `json:"max_output_tokens,omitempty"`
	Temperature        *float64        `json:"temperature,omitempty"`
	TopP               *float64        `json:"top_p,omitempty"`
	Stream             *bool           `json:"stream,omitempty"`
	PreviousResponseID *string         `json:"previous_response_id,omitempty"`
	Tools              []ResponsesTool `json:"tools,omitempty"`
	ToolChoice         any             `json:"tool_choice,omitempty"`
	User               *string         `json:"user,omitempty"`
	Metadata           map[string]any  `json:"metadata,omitempty"`
	ParallelToolCalls  *bool           `json:"parallel_tool_calls,omitempty"`
	Store              *bool           `json:"store,omitempty"`
}

type ResponsesTool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Strict      *bool          `json:"strict,omitempty"`
	Function    *ToolFunction  `json:"function,omitempty"`
}

type ResponsesResponse struct {
	ID                 string                `json:"id"`
	Object             string                `json:"object"`
	CreatedAt          int64                 `json:"created_at"`
	Status             string                `json:"status"`
	Model              string                `json:"model"`
	Output             []ResponsesOutputItem `json:"output"`
	OutputText         string                `json:"output_text"`
	Usage              *ResponsesUsage       `json:"usage,omitempty"`
	PreviousResponseID *string               `json:"previous_response_id,omitempty"`
	Error              any                   `json:"error"`
	IncompleteDetails  any                   `json:"incomplete_details"`
}

type ResponsesOutputItem struct {
	ID        string                 `json:"id,omitempty"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status,omitempty"`
	Role      string                 `json:"role,omitempty"`
	Content   []ResponsesContentPart `json:"content,omitempty"`
	CallID    string                 `json:"call_id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Arguments string                 `json:"arguments,omitempty"`
}

type ResponsesContentPart struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Annotations []any  `json:"annotations,omitempty"`
}

type ResponsesUsage struct {
	InputTokens              int  `json:"input_tokens"`
	OutputTokens             int  `json:"output_tokens"`
	TotalTokens              int  `json:"total_tokens"`
	InputTokensDetails       any  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails      any  `json:"output_tokens_details,omitempty"`
	CacheCreationInputTokens *int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     *int `json:"cache_read_input_tokens,omitempty"`
}
