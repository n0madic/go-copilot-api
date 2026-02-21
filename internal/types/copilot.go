package types

type ModelsResponse struct {
	Data   []Model `json:"data"`
	Object string  `json:"object"`
}

type Model struct {
	Capabilities      ModelCapabilities `json:"capabilities"`
	ID                string            `json:"id"`
	ModelPickerEnable bool              `json:"model_picker_enabled"`
	Name              string            `json:"name"`
	Object            string            `json:"object"`
	Preview           bool              `json:"preview"`
	Vendor            string            `json:"vendor"`
	Version           string            `json:"version"`
	Policy            *ModelPolicy      `json:"policy,omitempty"`
}

type ModelPolicy struct {
	State string `json:"state"`
	Terms string `json:"terms"`
}

type ModelCapabilities struct {
	Family    string        `json:"family"`
	Limits    ModelLimits   `json:"limits"`
	Object    string        `json:"object"`
	Supports  ModelSupports `json:"supports"`
	Tokenizer string        `json:"tokenizer"`
	Type      string        `json:"type"`
}

type ModelLimits struct {
	MaxContextWindowTokens *int `json:"max_context_window_tokens,omitempty"`
	MaxOutputTokens        *int `json:"max_output_tokens,omitempty"`
	MaxPromptTokens        *int `json:"max_prompt_tokens,omitempty"`
	MaxInputs              *int `json:"max_inputs,omitempty"`
}

type ModelSupports struct {
	ToolCalls         *bool `json:"tool_calls,omitempty"`
	ParallelToolCalls *bool `json:"parallel_tool_calls,omitempty"`
	Dimensions        *bool `json:"dimensions,omitempty"`
}
