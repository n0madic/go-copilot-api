package util_test

import (
	"testing"
	"time"

	"github.com/n0madic/go-copilot-api/internal/api"
	"github.com/n0madic/go-copilot-api/internal/state"
	"github.com/n0madic/go-copilot-api/internal/util"
)

func TestRateLimitRejectsWhenWaitDisabled(t *testing.T) {
	st := state.New()
	r := 1
	st.SetRateLimitSeconds(&r)
	st.SetRateLimitWait(false)
	st.SetLastRequestMS(time.Now().UnixMilli())

	err := util.CheckRateLimit(st)
	if err == nil {
		t.Fatalf("expected rate limit error")
	}
	httpErr, ok := err.(*api.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 429 {
		t.Fatalf("expected 429, got %d", httpErr.StatusCode)
	}
}

func TestRateLimitWaitsWhenEnabled(t *testing.T) {
	st := state.New()
	r := 1
	st.SetRateLimitSeconds(&r)
	st.SetRateLimitWait(true)
	st.SetLastRequestMS(time.Now().UnixMilli())

	start := time.Now()
	err := util.CheckRateLimit(st)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < 900*time.Millisecond {
		t.Fatalf("expected wait close to 1s, got %s", elapsed)
	}
}
