package copilot_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/n0madic/go-copilot-api/internal/copilot"
	"github.com/n0madic/go-copilot-api/internal/state"
	"github.com/n0madic/go-copilot-api/internal/types"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCreateChatCompletionsXInitiatorAgent(t *testing.T) {
	st := state.New()
	st.SetCopilotToken("test-token")
	st.SetVSCodeVersion("1.0.0")
	st.SetAccountType("individual")

	var gotHeader string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotHeader = req.Header.Get("X-Initiator")
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"id":"123","object":"chat.completion","choices":[]}`)),
			Header:     make(http.Header),
		}, nil
	})}

	copilotClient := copilot.NewClient(client, st)
	payload := types.ChatCompletionsPayload{
		Model: "gpt-test",
		Messages: []types.Message{
			{Role: "user", Content: "hi"},
			{Role: "tool", Content: "tool call"},
		},
	}
	resp, err := copilotClient.CreateChatCompletions(context.Background(), payload)
	if err != nil {
		t.Fatalf("CreateChatCompletions returned error: %v", err)
	}
	resp.Body.Close()

	if gotHeader != "agent" {
		t.Fatalf("expected X-Initiator=agent, got %q", gotHeader)
	}
}

func TestCreateChatCompletionsXInitiatorUser(t *testing.T) {
	st := state.New()
	st.SetCopilotToken("test-token")
	st.SetVSCodeVersion("1.0.0")
	st.SetAccountType("individual")

	var gotHeader string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotHeader = req.Header.Get("X-Initiator")
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"id":"123","object":"chat.completion","choices":[]}`)),
			Header:     make(http.Header),
		}, nil
	})}

	copilotClient := copilot.NewClient(client, st)
	payload := types.ChatCompletionsPayload{
		Model: "gpt-test",
		Messages: []types.Message{
			{Role: "user", Content: "hi"},
			{Role: "user", Content: "hello again"},
		},
	}
	resp, err := copilotClient.CreateChatCompletions(context.Background(), payload)
	if err != nil {
		t.Fatalf("CreateChatCompletions returned error: %v", err)
	}
	resp.Body.Close()

	if gotHeader != "user" {
		t.Fatalf("expected X-Initiator=user, got %q", gotHeader)
	}
}
