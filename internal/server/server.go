package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/n0madic/go-copilot-api/internal/anthropic"
	"github.com/n0madic/go-copilot-api/internal/api"
	"github.com/n0madic/go-copilot-api/internal/config"
	"github.com/n0madic/go-copilot-api/internal/copilot"
	gh "github.com/n0madic/go-copilot-api/internal/github"
	"github.com/n0madic/go-copilot-api/internal/sse"
	"github.com/n0madic/go-copilot-api/internal/state"
	"github.com/n0madic/go-copilot-api/internal/tokenizer"
	"github.com/n0madic/go-copilot-api/internal/types"
	"github.com/n0madic/go-copilot-api/internal/util"
)

type Server struct {
	state   *state.State
	copilot *copilot.Client
	github  *gh.Client
}

func New(st *state.State, copilotClient *copilot.Client, githubClient *gh.Client) *Server {
	return &Server{state: st, copilot: copilotClient, github: githubClient}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("/models", s.handleModels)
	mux.HandleFunc("/embeddings", s.handleEmbeddings)
	mux.HandleFunc("/usage", s.handleUsage)

	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/embeddings", s.handleEmbeddings)
	mux.HandleFunc("/v1/messages", s.handleMessages)
	mux.HandleFunc("/v1/messages/count_tokens", s.handleCountTokens)

	return withMiddleware(mux)
}

func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		lrw := &loggingResponseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(lrw, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.status,
			"duration", time.Since(start).String(),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(w.status)
	}
	return w.ResponseWriter.Write(b)
}

func (w *loggingResponseWriter) Flush() {
	flusher, ok := w.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}
	flusher.Flush()
}

func (w *loggingResponseWriter) HeaderWritten() bool {
	return w.wroteHeader
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	w.Header().Set("content-type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Server running"))
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	if err := util.CheckRateLimit(s.state); err != nil {
		s.forwardError(w, err)
		return
	}
	var payload types.ChatCompletionsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.forwardError(w, err)
		return
	}
	payload.Model = s.resolveModelID(payload.Model)

	selectedModel := s.findModel(payload.Model)
	if selectedModel != nil {
		count := tokenizer.GetTokenCount(payload, *selectedModel)
		slog.Info("Current token count", "input", count.Input, "output", count.Output)
	}

	if payload.MaxTokens == nil && selectedModel != nil && selectedModel.Capabilities.Limits.MaxOutputTokens != nil {
		v := *selectedModel.Capabilities.Limits.MaxOutputTokens
		payload.MaxTokens = &v
	}

	ctx := r.Context()
	resp, err := s.copilot.CreateChatCompletions(ctx, payload)
	if err != nil {
		s.forwardError(w, err)
		return
	}
	defer resp.Body.Close()

	stream := payload.Stream != nil && *payload.Stream
	if !stream {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, resp.Body)
		return
	}

	if err := proxySSE(w, resp); err != nil {
		s.forwardError(w, err)
		return
	}
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	if s.state.Models() == nil {
		if err := copilot.CacheModels(r.Context(), s.copilot, s.state); err != nil {
			s.forwardError(w, err)
			return
		}
	}
	models := s.state.Models()
	if models == nil {
		s.forwardError(w, errors.New("models not available"))
		return
	}
	type apiModel struct {
		ID          string `json:"id"`
		Object      string `json:"object"`
		Type        string `json:"type"`
		Created     int64  `json:"created"`
		CreatedAt   string `json:"created_at"`
		OwnedBy     string `json:"owned_by"`
		DisplayName string `json:"display_name"`
	}
	result := make([]apiModel, 0, len(models.Data))
	for _, model := range models.Data {
		result = append(result, apiModel{
			ID:          model.ID,
			Object:      "model",
			Type:        "model",
			Created:     0,
			CreatedAt:   time.Unix(0, 0).UTC().Format(time.RFC3339),
			OwnedBy:     model.Vendor,
			DisplayName: model.Name,
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": result, "has_more": false})
}

func (s *Server) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	var payload types.EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.forwardError(w, err)
		return
	}
	response, err := s.copilot.CreateEmbeddings(r.Context(), payload)
	if err != nil {
		s.forwardError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	usage, err := s.github.GetCopilotUsage(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Failed to fetch Copilot usage"})
		return
	}
	s.writeJSON(w, http.StatusOK, usage)
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	if err := util.CheckRateLimit(s.state); err != nil {
		s.forwardError(w, err)
		return
	}
	var anthropicPayload types.AnthropicMessagesPayload
	if err := json.NewDecoder(r.Body).Decode(&anthropicPayload); err != nil {
		s.forwardError(w, err)
		return
	}
	openAIPayload := anthropic.TranslateToOpenAI(anthropicPayload)
	openAIPayload.Model = s.resolveModelID(openAIPayload.Model)

	resp, err := s.copilot.CreateChatCompletions(r.Context(), openAIPayload)
	if err != nil {
		s.forwardError(w, err)
		return
	}
	defer resp.Body.Close()

	stream := openAIPayload.Stream != nil && *openAIPayload.Stream
	if !stream {
		var completion types.ChatCompletionResponse
		if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
			s.forwardError(w, err)
			return
		}
		translated := anthropic.TranslateToAnthropic(completion)
		s.writeJSON(w, http.StatusOK, translated)
		return
	}

	if err := translateAnthropicStream(w, resp); err != nil {
		s.forwardError(w, err)
		return
	}
}

func (s *Server) handleCountTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}
	anthropicBeta := r.Header.Get("anthropic-beta")
	var payload types.AnthropicMessagesPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	openAIPayload := anthropic.TranslateToOpenAI(payload)
	openAIPayload.Model = s.resolveModelID(openAIPayload.Model)
	selectedModel := s.findModel(payload.Model)
	if selectedModel == nil {
		slog.Warn("model not found, returning default token count", "model", payload.Model)
		s.writeJSON(w, http.StatusOK, map[string]any{"input_tokens": 1})
		return
	}

	count := tokenizer.GetTokenCount(openAIPayload, *selectedModel)
	if len(payload.Tools) > 0 {
		mcpToolExists := false
		if strings.HasPrefix(anthropicBeta, "claude-code") {
			for _, tool := range payload.Tools {
				if strings.HasPrefix(tool.Name, "mcp__") {
					mcpToolExists = true
					break
				}
			}
		}
		if !mcpToolExists {
			if strings.HasPrefix(payload.Model, "claude") {
				count.Input += 346
			} else if strings.HasPrefix(payload.Model, "grok") {
				count.Input += 480
			}
		}
	}

	finalTokenCount := count.Input + count.Output
	if strings.HasPrefix(payload.Model, "claude") {
		finalTokenCount = int(float64(finalTokenCount)*1.15 + 0.5)
	} else if strings.HasPrefix(payload.Model, "grok") {
		finalTokenCount = int(float64(finalTokenCount)*1.03 + 0.5)
	}

	slog.Info("Token count", "input_tokens", finalTokenCount)
	s.writeJSON(w, http.StatusOK, map[string]any{"input_tokens": finalTokenCount})
}

func proxySSE(w http.ResponseWriter, resp *http.Response) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("streaming unsupported")
	}
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	return sse.ReadEvents(resp.Body, func(ev sse.Event) error {
		if ev.Event != "" {
			if _, err := io.WriteString(w, "event: "+ev.Event+"\n"); err != nil {
				return err
			}
		}
		for _, line := range strings.Split(ev.Data, "\n") {
			if line == "" {
				continue
			}
			if _, err := io.WriteString(w, "data: "+line+"\n"); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})
}

func translateAnthropicStream(w http.ResponseWriter, resp *http.Response) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("streaming unsupported")
	}
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	streamState := &types.AnthropicStreamState{
		MessageStartSent:  false,
		ContentBlockIndex: 0,
		ContentBlockOpen:  false,
		ToolCalls:         map[int]types.ToolCallState{},
	}

	return sse.ReadEvents(resp.Body, func(ev sse.Event) error {
		data := strings.TrimSpace(ev.Data)
		if data == "" {
			return nil
		}
		if data == "[DONE]" {
			return nil
		}
		var chunk types.ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return err
		}
		events := anthropic.TranslateChunkToAnthropicEvents(chunk, streamState)
		for _, event := range events {
			eventType, _ := event["type"].(string)
			if _, err := io.WriteString(w, "event: "+eventType+"\n"); err != nil {
				return err
			}
			payload, _ := json.Marshal(event)
			if _, err := io.WriteString(w, "data: "+string(payload)+"\n\n"); err != nil {
				return err
			}
			flusher.Flush()
		}
		return nil
	})
}

func (s *Server) findModel(id string) *types.Model {
	models := s.state.Models()
	if models == nil {
		return nil
	}
	resolved := s.resolveModelID(id)
	for _, m := range models.Data {
		if m.ID == resolved {
			copy := m
			return &copy
		}
	}
	return nil
}

func (s *Server) resolveModelID(id string) string {
	if id == "" {
		return id
	}
	// Use the pre-built model ID set from State for O(1) lookups.
	available := s.state.ModelIDSet()
	if available == nil {
		return id
	}
	for _, candidate := range modelCandidates(id) {
		if _, ok := available[candidate]; ok {
			return candidate
		}
	}
	return id
}

func modelCandidates(id string) []string {
	out := make([]string, 0, 6)
	seen := make(map[string]struct{}, 6)
	add := func(v string) {
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	add(id)

	for _, prefix := range []string{"claude-sonnet-4-", "claude-opus-4-", "claude-haiku-4-"} {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		base := strings.TrimSuffix(prefix, "-")
		tail := strings.TrimPrefix(id, prefix)
		add(base)
		if tail == "" {
			continue
		}
		add(base + "." + tail)
		if idx := strings.IndexByte(tail, '-'); idx > 0 {
			majorMinor := tail[:idx]
			if util.IsDigits(majorMinor) {
				add(base + "." + majorMinor)
			}
		}
		if util.IsDigits(tail) && len(tail) >= 6 {
			// date-like suffix, keep family base candidate
			add(base)
		}
	}

	return out
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Warn("failed to encode JSON response", "error", err)
	}
}

func (s *Server) forwardError(w http.ResponseWriter, err error) {
	slog.Error("Error occurred", "error", err)
	if hw, ok := w.(interface{ HeaderWritten() bool }); ok && hw.HeaderWritten() {
		return
	}
	status := http.StatusInternalServerError
	message := err.Error()
	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) {
		if httpErr.StatusCode > 0 {
			status = httpErr.StatusCode
		}
		if len(httpErr.Body) > 0 {
			message = string(httpErr.Body)
		}
	}
	s.writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    "error",
		},
	})
}

func ParsePort(v string) int {
	if v == "" {
		return config.DefaultPort
	}
	port, err := strconv.Atoi(v)
	if err != nil || port <= 0 || port > 65535 {
		return config.DefaultPort
	}
	return port
}

func Shutdown(ctx context.Context, srv *http.Server) error {
	return srv.Shutdown(ctx)
}
