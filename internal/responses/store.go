package responses

import (
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/n0madic/go-copilot-api/internal/types"
)

const (
	DefaultStoreTTL      = 30 * time.Minute
	DefaultStoreCapacity = 1024
)

type Store struct {
	mu         sync.Mutex
	ttl        time.Duration
	maxEntries int
	entries    map[string]storeEntry
}

type storeEntry struct {
	messages  []types.Message
	createdAt time.Time
	expiresAt time.Time
}

func NewStore(ttl time.Duration, maxEntries int) *Store {
	if ttl <= 0 {
		ttl = DefaultStoreTTL
	}
	if maxEntries <= 0 {
		maxEntries = DefaultStoreCapacity
	}
	return &Store{
		ttl:        ttl,
		maxEntries: maxEntries,
		entries:    make(map[string]storeEntry, maxEntries),
	}
}

func (s *Store) Get(responseID string) ([]types.Message, bool) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.evictExpiredLocked(now)
	entry, ok := s.entries[responseID]
	if !ok {
		return nil, false
	}
	return cloneMessages(entry.messages), true
}

func (s *Store) Put(responseID string, messages []types.Message) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.evictExpiredLocked(now)
	s.entries[responseID] = storeEntry{
		messages:  cloneMessages(messages),
		createdAt: now,
		expiresAt: now.Add(s.ttl),
	}
	s.evictOverflowLocked()
}

func (s *Store) evictExpiredLocked(now time.Time) {
	for key, entry := range s.entries {
		if now.After(entry.expiresAt) {
			delete(s.entries, key)
		}
	}
}

func (s *Store) evictOverflowLocked() {
	overflow := len(s.entries) - s.maxEntries
	if overflow <= 0 {
		return
	}

	type candidate struct {
		key       string
		createdAt time.Time
	}
	candidates := make([]candidate, 0, len(s.entries))
	for key, entry := range s.entries {
		candidates = append(candidates, candidate{key: key, createdAt: entry.createdAt})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].createdAt.Before(candidates[j].createdAt)
	})

	for i := 0; i < overflow; i++ {
		delete(s.entries, candidates[i].key)
	}
}

func cloneMessages(messages []types.Message) []types.Message {
	if len(messages) == 0 {
		return nil
	}
	buf, err := json.Marshal(messages)
	if err != nil {
		out := make([]types.Message, len(messages))
		copy(out, messages)
		return out
	}
	var out []types.Message
	if err := json.Unmarshal(buf, &out); err != nil {
		out = make([]types.Message, len(messages))
		copy(out, messages)
	}
	return out
}
