package responses

import (
	"sort"
	"strings"
	"time"

	"github.com/n0madic/go-copilot-api/internal/types"
)

type StreamEvent struct {
	Name    string
	Payload map[string]any
}

type StreamTranslator struct {
	responseID         string
	model              string
	previousResponseID *string
	createdAt          int64

	messageOutputIndex int
	messageText        strings.Builder

	outputItems []types.ResponsesOutputItem
	toolCalls   map[int]*toolCallState

	usage     *types.ResponsesUsage
	completed bool
}

type toolCallState struct {
	index       int
	outputIndex int
	itemID      string
	callID      string
	name        string
	arguments   strings.Builder
	added       bool
	done        bool
}

func NewStreamTranslator(responseID string, model string, previousResponseID *string, createdAt int64) *StreamTranslator {
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}
	return &StreamTranslator{
		responseID:         responseID,
		model:              model,
		previousResponseID: previousResponseID,
		createdAt:          createdAt,
		messageOutputIndex: -1,
		toolCalls:          make(map[int]*toolCallState),
	}
}

func (t *StreamTranslator) CreatedEvent() StreamEvent {
	response := types.ResponsesResponse{
		ID:                 t.responseID,
		Object:             "response",
		CreatedAt:          t.createdAt,
		Status:             "in_progress",
		Model:              t.model,
		Output:             []types.ResponsesOutputItem{},
		OutputText:         "",
		PreviousResponseID: t.previousResponseID,
		Error:              nil,
		IncompleteDetails:  nil,
	}
	return t.newEvent("response.created", map[string]any{"response": response})
}

func (t *StreamTranslator) HandleChunk(chunk types.ChatCompletionChunk) []StreamEvent {
	if chunk.Model != "" {
		t.model = chunk.Model
	}
	if chunk.Usage != nil {
		t.usage = mapUsage(chunk.Usage)
	}
	if len(chunk.Choices) == 0 {
		return nil
	}

	choice := chunk.Choices[0]
	delta := choice.Delta
	events := make([]StreamEvent, 0, 8)

	if delta.Content != nil && *delta.Content != "" {
		events = append(events, t.ensureMessageStarted()...)
		t.messageText.WriteString(*delta.Content)
		text := t.messageText.String()
		t.outputItems[t.messageOutputIndex].Content[0].Text = text
		events = append(events, t.newEvent("response.output_text.delta", map[string]any{
			"response_id":   t.responseID,
			"output_index":  t.messageOutputIndex,
			"content_index": 0,
			"delta":         *delta.Content,
		}))
	}

	for _, toolCall := range delta.ToolCalls {
		state := t.ensureToolCallState(toolCall.Index)
		if toolCall.ID != "" {
			state.callID = toolCall.ID
		}
		if toolCall.Function != nil && toolCall.Function.Name != "" {
			state.name = toolCall.Function.Name
		}

		if !state.added && (state.callID != "" || state.name != "" || (toolCall.Function != nil && toolCall.Function.Arguments != "")) {
			events = append(events, t.addToolCallItem(state))
		}

		if toolCall.Function != nil && toolCall.Function.Arguments != "" {
			if !state.added {
				events = append(events, t.addToolCallItem(state))
			}
			state.arguments.WriteString(toolCall.Function.Arguments)
			args := state.arguments.String()
			t.outputItems[state.outputIndex].Arguments = args
			events = append(events, t.newEvent("response.function_call_arguments.delta", map[string]any{
				"response_id":  t.responseID,
				"output_index": state.outputIndex,
				"item_id":      state.itemID,
				"delta":        toolCall.Function.Arguments,
			}))
		}
	}

	if choice.FinishReason != nil {
		events = append(events, t.finalizeEvents()...)
	}

	return events
}

func (t *StreamTranslator) ForceComplete() []StreamEvent {
	if t.completed {
		return nil
	}
	return t.finalizeEvents()
}

func (t *StreamTranslator) Completed() bool {
	return t.completed
}

func (t *StreamTranslator) AssistantMessage() types.Message {
	toolCalls := make([]types.ToolCall, 0)
	for _, item := range t.outputItems {
		if item.Type != "function_call" {
			continue
		}
		toolCalls = append(toolCalls, types.ToolCall{
			ID:   item.CallID,
			Type: "function",
			Function: types.ToolCallFn{
				Name:      item.Name,
				Arguments: item.Arguments,
			},
		})
	}

	return types.Message{
		Role:      "assistant",
		Content:   t.messageText.String(),
		ToolCalls: toolCalls,
	}
}

func (t *StreamTranslator) ensureMessageStarted() []StreamEvent {
	if t.messageOutputIndex >= 0 {
		return nil
	}

	item := types.ResponsesOutputItem{
		ID:     newMessageID(),
		Type:   "message",
		Status: "in_progress",
		Role:   "assistant",
		Content: []types.ResponsesContentPart{{
			Type:        "output_text",
			Text:        "",
			Annotations: []any{},
		}},
	}

	t.messageOutputIndex = len(t.outputItems)
	t.outputItems = append(t.outputItems, item)

	return []StreamEvent{
		t.newEvent("response.output_item.added", map[string]any{
			"response_id":  t.responseID,
			"output_index": t.messageOutputIndex,
			"item":         item,
		}),
		t.newEvent("response.content_part.added", map[string]any{
			"response_id":   t.responseID,
			"output_index":  t.messageOutputIndex,
			"content_index": 0,
			"part":          item.Content[0],
		}),
	}
}

func (t *StreamTranslator) ensureToolCallState(index int) *toolCallState {
	if state, ok := t.toolCalls[index]; ok {
		return state
	}
	state := &toolCallState{index: index}
	t.toolCalls[index] = state
	return state
}

func (t *StreamTranslator) addToolCallItem(state *toolCallState) StreamEvent {
	if state.callID == "" {
		state.callID = newCallID()
	}
	if state.name == "" {
		state.name = "function"
	}

	item := types.ResponsesOutputItem{
		ID:        newFunctionID(),
		Type:      "function_call",
		Status:    "in_progress",
		CallID:    state.callID,
		Name:      state.name,
		Arguments: state.arguments.String(),
	}

	state.outputIndex = len(t.outputItems)
	state.itemID = item.ID
	state.added = true
	t.outputItems = append(t.outputItems, item)

	return t.newEvent("response.output_item.added", map[string]any{
		"response_id":  t.responseID,
		"output_index": state.outputIndex,
		"item":         item,
	})
}

func (t *StreamTranslator) finalizeEvents() []StreamEvent {
	if t.completed {
		return nil
	}

	events := make([]StreamEvent, 0, 8)

	if t.messageOutputIndex >= 0 {
		t.outputItems[t.messageOutputIndex].Status = "completed"
		events = append(events,
			t.newEvent("response.output_text.done", map[string]any{
				"response_id":   t.responseID,
				"output_index":  t.messageOutputIndex,
				"content_index": 0,
				"text":          t.messageText.String(),
			}),
			t.newEvent("response.output_item.done", map[string]any{
				"response_id":  t.responseID,
				"output_index": t.messageOutputIndex,
				"item":         t.outputItems[t.messageOutputIndex],
			}),
		)
	}

	for _, state := range t.sortedToolCalls() {
		if !state.added || state.done {
			continue
		}
		args := state.arguments.String()
		t.outputItems[state.outputIndex].Arguments = args
		t.outputItems[state.outputIndex].Status = "completed"
		state.done = true

		events = append(events,
			t.newEvent("response.function_call_arguments.done", map[string]any{
				"response_id":  t.responseID,
				"output_index": state.outputIndex,
				"item_id":      state.itemID,
				"arguments":    args,
			}),
			t.newEvent("response.output_item.done", map[string]any{
				"response_id":  t.responseID,
				"output_index": state.outputIndex,
				"item":         t.outputItems[state.outputIndex],
			}),
		)
	}

	completedResponse := types.ResponsesResponse{
		ID:                 t.responseID,
		Object:             "response",
		CreatedAt:          t.createdAt,
		Status:             "completed",
		Model:              t.model,
		Output:             append([]types.ResponsesOutputItem(nil), t.outputItems...),
		OutputText:         t.messageText.String(),
		Usage:              t.usage,
		PreviousResponseID: t.previousResponseID,
		Error:              nil,
		IncompleteDetails:  nil,
	}

	events = append(events, t.newEvent("response.completed", map[string]any{"response": completedResponse}))
	t.completed = true
	return events
}

func (t *StreamTranslator) sortedToolCalls() []*toolCallState {
	states := make([]*toolCallState, 0, len(t.toolCalls))
	for _, state := range t.toolCalls {
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool {
		if states[i].outputIndex == states[j].outputIndex {
			return states[i].index < states[j].index
		}
		return states[i].outputIndex < states[j].outputIndex
	})
	return states
}

func (t *StreamTranslator) newEvent(name string, payload map[string]any) StreamEvent {
	if payload == nil {
		payload = map[string]any{}
	}
	payload["type"] = name
	return StreamEvent{Name: name, Payload: payload}
}
