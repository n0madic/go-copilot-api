package util

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/n0madic/go-copilot-api/internal/api"
	"github.com/n0madic/go-copilot-api/internal/state"
)

func CheckRateLimit(st *state.State) error {
	rateLimit := st.RateLimitSeconds()
	if rateLimit == nil {
		return nil
	}

	limitMS := int64(*rateLimit) * 1000

	for {
		nowMS := time.Now().UnixMilli()
		allowed, waitMS := st.CheckAndUpdateRateLimit(nowMS, limitMS)
		if allowed {
			return nil
		}

		// Ceiling division: round up to the nearest full second.
		waitSeconds := int((waitMS + 999) / 1000)
		if !st.RateLimitWait() {
			slog.Warn("rate limit exceeded", "wait_seconds", waitSeconds)
			return &api.HTTPError{
				Message:    "Rate limit exceeded",
				StatusCode: 429,
				Body:       []byte(`{"message":"Rate limit exceeded"}`),
			}
		}

		slog.Warn("rate limit reached, waiting", "wait_seconds", waitSeconds)
		time.Sleep(time.Duration(waitMS) * time.Millisecond)
		slog.Info("rate limit wait completed")
	}
}

func HumanizeQuotaUsed(total float64, remaining float64, percentRemaining float64) string {
	used := total - remaining
	percentUsed := 0.0
	if total > 0 {
		percentUsed = used / total * 100
	}
	return fmt.Sprintf("%.0f/%.0f used (%.1f%% used, %.1f%% remaining)", used, total, percentUsed, percentRemaining)
}
