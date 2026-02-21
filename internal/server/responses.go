package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/n0madic/go-copilot-api/internal/responses"
	"github.com/n0madic/go-copilot-api/internal/sse"
	"github.com/n0madic/go-copilot-api/internal/types"
	"github.com/n0madic/go-copilot-api/internal/util"
)

func (s *Server) handleResponses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	if err := util.CheckRateLimit(s.state); err != nil {
		s.forwardError(w, err)
		return
	}

	var req types.ResponsesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeResponsesInvalidRequest(w, "invalid JSON body", "")
		return
	}

	if strings.TrimSpace(req.Model) == "" {
		s.writeResponsesInvalidRequest(w, "model is required", "model")
		return
	}

	var history []types.Message
	if req.PreviousResponseID != nil {
		id := strings.TrimSpace(*req.PreviousResponseID)
		if id != "" {
			resolvedHistory, ok := s.responseStore.Get(id)
			if !ok {
				s.writeJSON(w, http.StatusBadRequest, map[string]any{
					"error": map[string]any{
						"message": "unknown or expired previous_response_id",
						"type":    "invalid_request_error",
						"param":   "previous_response_id",
						"code":    "invalid_previous_response_id",
					},
				})
				return
			}
			history = resolvedHistory
			req.PreviousResponseID = &id
		}
	}

	payload, err := responses.BuildChatCompletionsPayload(req, history)
	if err != nil {
		s.writeResponsesInvalidRequest(w, err.Error(), "input")
		return
	}

	payload.Model = s.resolveModelID(payload.Model)
	selectedModel := s.findModel(payload.Model)
	if payload.MaxTokens == nil && selectedModel != nil && selectedModel.Capabilities.Limits.MaxOutputTokens != nil {
		v := *selectedModel.Capabilities.Limits.MaxOutputTokens
		payload.MaxTokens = &v
	}

	upstreamResp, err := s.copilot.CreateChatCompletions(r.Context(), payload)
	if err != nil {
		s.forwardError(w, err)
		return
	}
	defer upstreamResp.Body.Close()

	responseID := responses.NewResponseID()
	stream := req.Stream != nil && *req.Stream
	if !stream {
		var completion types.ChatCompletionResponse
		if err := json.NewDecoder(upstreamResp.Body).Decode(&completion); err != nil {
			s.forwardError(w, err)
			return
		}

		translated, assistant := responses.CompletionToResponse(req, completion, responseID)
		conversation := make([]types.Message, 0, len(payload.Messages)+1)
		conversation = append(conversation, payload.Messages...)
		conversation = append(conversation, assistant)
		s.responseStore.Put(responseID, conversation)
		s.writeJSON(w, http.StatusOK, translated)
		return
	}

	if err := s.translateResponsesStream(w, upstreamResp, req, responseID, payload.Messages, payload.Model); err != nil {
		s.forwardError(w, err)
		return
	}
}

func (s *Server) translateResponsesStream(
	w http.ResponseWriter,
	upstreamResp *http.Response,
	req types.ResponsesRequest,
	responseID string,
	requestMessages []types.Message,
	model string,
) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("streaming unsupported")
	}

	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	translator := responses.NewStreamTranslator(responseID, model, req.PreviousResponseID, time.Now().Unix())
	if err := writeResponsesSSEEvent(w, flusher, translator.CreatedEvent()); err != nil {
		return err
	}

	if err := sse.ReadEvents(upstreamResp.Body, func(ev sse.Event) error {
		data := strings.TrimSpace(ev.Data)
		if data == "" {
			return nil
		}
		if data == "[DONE]" {
			for _, outEvent := range translator.ForceComplete() {
				if err := writeResponsesSSEEvent(w, flusher, outEvent); err != nil {
					return err
				}
			}
			return nil
		}

		var chunk types.ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return err
		}
		for _, outEvent := range translator.HandleChunk(chunk) {
			if err := writeResponsesSSEEvent(w, flusher, outEvent); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	for _, outEvent := range translator.ForceComplete() {
		if err := writeResponsesSSEEvent(w, flusher, outEvent); err != nil {
			return err
		}
	}

	conversation := make([]types.Message, 0, len(requestMessages)+1)
	conversation = append(conversation, requestMessages...)
	conversation = append(conversation, translator.AssistantMessage())
	s.responseStore.Put(responseID, conversation)
	return nil
}

func writeResponsesSSEEvent(w io.Writer, flusher http.Flusher, ev responses.StreamEvent) error {
	if ev.Name != "" {
		if _, err := io.WriteString(w, "event: "+ev.Name+"\n"); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(ev.Payload)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, "data: "+string(payload)+"\n\n"); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func (s *Server) writeResponsesInvalidRequest(w http.ResponseWriter, message string, param string) {
	errPayload := map[string]any{
		"message": message,
		"type":    "invalid_request_error",
	}
	if param != "" {
		errPayload["param"] = param
	}
	s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": errPayload})
}
