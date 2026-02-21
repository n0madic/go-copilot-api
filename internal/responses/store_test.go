package responses

import (
	"testing"
	"time"

	"github.com/n0madic/go-copilot-api/internal/types"
)

func TestStorePutGetRoundTrip(t *testing.T) {
	store := NewStore(time.Minute, 10)
	messages := []types.Message{{Role: "user", Content: "hello"}}

	store.Put("resp_1", messages)

	got, ok := store.Get("resp_1")
	if !ok {
		t.Fatalf("expected store hit")
	}
	if len(got) != 1 || got[0].Role != "user" || got[0].Content != "hello" {
		t.Fatalf("unexpected messages: %#v", got)
	}

	messages[0].Role = "assistant"
	gotAgain, ok := store.Get("resp_1")
	if !ok {
		t.Fatalf("expected store hit")
	}
	if gotAgain[0].Role != "user" {
		t.Fatalf("expected cloned storage, got role=%q", gotAgain[0].Role)
	}
}

func TestStoreExpiresEntries(t *testing.T) {
	store := NewStore(20*time.Millisecond, 10)
	store.Put("resp_1", []types.Message{{Role: "user", Content: "hello"}})

	time.Sleep(40 * time.Millisecond)
	if _, ok := store.Get("resp_1"); ok {
		t.Fatalf("expected entry to expire")
	}
}

func TestStoreEvictsOldestOnCapacity(t *testing.T) {
	store := NewStore(time.Minute, 2)
	store.Put("resp_1", []types.Message{{Role: "user", Content: "one"}})
	store.Put("resp_2", []types.Message{{Role: "user", Content: "two"}})
	store.Put("resp_3", []types.Message{{Role: "user", Content: "three"}})

	if _, ok := store.Get("resp_1"); ok {
		t.Fatalf("expected oldest entry to be evicted")
	}
	if _, ok := store.Get("resp_2"); !ok {
		t.Fatalf("expected resp_2 to exist")
	}
	if _, ok := store.Get("resp_3"); !ok {
		t.Fatalf("expected resp_3 to exist")
	}
}
