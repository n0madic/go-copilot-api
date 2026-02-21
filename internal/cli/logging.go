package cli

import (
	"log/slog"
	"os"
	"runtime"
	"strings"
)

func setLogger(level slog.Level) {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(h))
}

func runtimeVersion() string {
	return strings.TrimPrefix(runtime.Version(), "go")
}
