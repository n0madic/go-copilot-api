package state

import (
	"sync"

	"github.com/n0madic/go-copilot-api/internal/types"
)

type State struct {
	mu sync.RWMutex

	githubToken  string
	copilotToken string

	accountType string
	models      *types.ModelsResponse
	modelIDSet  map[string]struct{}
	vscodeVer   string

	rateLimitWait bool
	showToken     bool

	rateLimitSeconds *int
	lastRequestMS    int64
}

func New() *State {
	return &State{accountType: "individual"}
}

func (s *State) SetGitHubToken(v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.githubToken = v
}

func (s *State) GitHubToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.githubToken
}

func (s *State) SetCopilotToken(v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.copilotToken = v
}

func (s *State) CopilotToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.copilotToken
}

func (s *State) SetAccountType(v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountType = v
}

func (s *State) AccountType() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accountType
}

// SetModels stores the models response and rebuilds the model ID set for O(1) lookups.
func (s *State) SetModels(v *types.ModelsResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models = v
	if v == nil {
		s.modelIDSet = nil
		return
	}
	set := make(map[string]struct{}, len(v.Data))
	for _, m := range v.Data {
		set[m.ID] = struct{}{}
	}
	s.modelIDSet = set
}

func (s *State) Models() *types.ModelsResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.models
}

// ModelIDSet returns the pre-built model ID set. Callers must not mutate the map.
func (s *State) ModelIDSet() map[string]struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modelIDSet
}

func (s *State) SetVSCodeVersion(v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vscodeVer = v
}

func (s *State) VSCodeVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vscodeVer
}

func (s *State) SetRateLimitWait(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rateLimitWait = v
}

func (s *State) RateLimitWait() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rateLimitWait
}

func (s *State) SetShowToken(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.showToken = v
}

func (s *State) ShowToken() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showToken
}

func (s *State) SetRateLimitSeconds(v *int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rateLimitSeconds = v
}

func (s *State) RateLimitSeconds() *int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.rateLimitSeconds == nil {
		return nil
	}
	copy := *s.rateLimitSeconds
	return &copy
}

func (s *State) SetLastRequestMS(v int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastRequestMS = v
}

func (s *State) LastRequestMS() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastRequestMS
}

// CheckAndUpdateRateLimit atomically checks whether the rate limit interval has
// passed and, if so, records the new timestamp.  It returns (true, 0) when the
// request is allowed, or (false, waitMS) when the caller must wait.
func (s *State) CheckAndUpdateRateLimit(nowMS int64, limitMS int64) (bool, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	elapsed := nowMS - s.lastRequestMS
	if elapsed < limitMS {
		return false, limitMS - elapsed
	}
	s.lastRequestMS = nowMS
	return true, 0
}
