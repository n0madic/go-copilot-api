package sse

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestReadEventsParsesBasicEvent(t *testing.T) {
	input := "event: message\ndata: hello\nid: 1\n\n"
	events := make([]Event, 0, 1)
	err := ReadEvents(strings.NewReader(input), func(ev Event) error {
		events = append(events, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("ReadEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != "message" {
		t.Fatalf("unexpected event name: %q", events[0].Event)
	}
	if events[0].Data != "hello" {
		t.Fatalf("unexpected data: %q", events[0].Data)
	}
	if events[0].ID != "1" {
		t.Fatalf("unexpected id: %q", events[0].ID)
	}
}

func TestReadEventsParsesMultilineDataAndIgnoresComments(t *testing.T) {
	input := ": heartbeat\nevent: update\ndata: line-1\ndata: line-2\nretry: 1000\n\n"
	events := make([]Event, 0, 1)
	err := ReadEvents(strings.NewReader(input), func(ev Event) error {
		events = append(events, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("ReadEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != "update" {
		t.Fatalf("unexpected event name: %q", events[0].Event)
	}
	if events[0].Data != "line-1\nline-2" {
		t.Fatalf("unexpected multiline data: %q", events[0].Data)
	}
}

func TestReadEventsFlushesLastEventWithoutTrailingBlankLine(t *testing.T) {
	input := "event: done\ndata: completed"
	events := make([]Event, 0, 1)
	err := ReadEvents(strings.NewReader(input), func(ev Event) error {
		events = append(events, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("ReadEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != "done" || events[0].Data != "completed" {
		t.Fatalf("unexpected event: %#v", events[0])
	}
}

func TestReadEventsPropagatesCallbackError(t *testing.T) {
	input := "data: one\n\ndata: two\n\n"
	wantErr := errors.New("callback failed")
	count := 0
	err := ReadEvents(strings.NewReader(input), func(ev Event) error {
		count++
		if count == 2 {
			return wantErr
		}
		return nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected callback error %v, got %v", wantErr, err)
	}
	if count != 2 {
		t.Fatalf("expected callback to run twice, got %d", count)
	}
}

func TestReadEventsSupportsLargeLineWithinBufferLimit(t *testing.T) {
	large := strings.Repeat("x", 256*1024)
	input := "data: " + large + "\n\n"
	events := make([]Event, 0, 1)
	err := ReadEvents(strings.NewReader(input), func(ev Event) error {
		events = append(events, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("ReadEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if len(events[0].Data) != len(large) {
		t.Fatalf("unexpected data length: got=%d want=%d", len(events[0].Data), len(large))
	}
}

func TestReadEventsReturnsScannerErrorWhenLineTooLong(t *testing.T) {
	tooLarge := strings.Repeat("x", 1024*1024+16)
	input := "data: " + tooLarge + "\n\n"
	err := ReadEvents(strings.NewReader(input), func(ev Event) error {
		return nil
	})
	if err == nil {
		t.Fatalf("expected scanner error for oversized token")
	}
	if !strings.Contains(err.Error(), "token too long") {
		t.Fatalf("expected token-too-long error, got %v", err)
	}
}

func TestReadEventsTrimsOnlySingleTrailingDataNewline(t *testing.T) {
	input := "data: one\ndata: two\n\n"
	var got Event
	err := ReadEvents(strings.NewReader(input), func(ev Event) error {
		got = ev
		return nil
	})
	if err != nil {
		t.Fatalf("ReadEvents returned error: %v", err)
	}
	want := "one\ntwo"
	if got.Data != want {
		t.Fatalf("unexpected data: got=%q want=%q", got.Data, want)
	}
}

func ExampleReadEvents() {
	input := "event: message\ndata: hello\n\n"
	_ = ReadEvents(strings.NewReader(input), func(ev Event) error {
		fmt.Printf("event=%s data=%s\n", ev.Event, ev.Data)
		return nil
	})
	// Output:
	// event=message data=hello
}
