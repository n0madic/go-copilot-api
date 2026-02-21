package cli

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
)

type envVars map[string]string

func isSafeEnvChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') ||
		(r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') ||
		r == '.' || r == '_' || r == '/' || r == ':' || r == '@' || r == '-'
}

func sanitizeEnvValue(v string) (string, error) {
	for _, r := range v {
		if !isSafeEnvChar(r) {
			return "", fmt.Errorf("unsafe character %q in env var value", r)
		}
	}
	return v, nil
}

func generateEnvScript(vars envVars, command string) string {
	shellName := detectShell()
	pairs := make([][2]string, 0, len(vars))
	for k, v := range vars {
		if v == "" {
			continue
		}
		safe, err := sanitizeEnvValue(v)
		if err != nil {
			slog.Warn("skipping env var with unsafe value", "key", k, "error", err)
			continue
		}
		pairs = append(pairs, [2]string{k, safe})
	}

	commandBlock := ""
	switch shellName {
	case "powershell":
		parts := make([]string, 0, len(pairs))
		for _, p := range pairs {
			parts = append(parts, "$env:"+p[0]+" = \""+p[1]+"\"")
		}
		commandBlock = strings.Join(parts, "; ")
	case "cmd":
		parts := make([]string, 0, len(pairs))
		for _, p := range pairs {
			parts = append(parts, "set "+p[0]+"="+p[1])
		}
		commandBlock = strings.Join(parts, " & ")
	case "fish":
		parts := make([]string, 0, len(pairs))
		for _, p := range pairs {
			parts = append(parts, "set -gx "+p[0]+" \""+p[1]+"\"")
		}
		commandBlock = strings.Join(parts, "; ")
	default:
		parts := make([]string, 0, len(pairs))
		for _, p := range pairs {
			parts = append(parts, p[0]+"=\""+p[1]+"\"")
		}
		if len(parts) > 0 {
			commandBlock = "export " + strings.Join(parts, " ")
		}
	}

	if commandBlock != "" && command != "" {
		sep := " && "
		if shellName == "cmd" {
			sep = " & "
		}
		return commandBlock + sep + command
	}
	if commandBlock != "" {
		return commandBlock
	}
	return command
}

func detectShell() string {
	if runtime.GOOS == "windows" {
		if strings.Contains(strings.ToLower(os.Getenv("COMSPEC")), "powershell") {
			return "powershell"
		}
		return "cmd"
	}
	sh := os.Getenv("SHELL")
	switch {
	case strings.HasSuffix(sh, "zsh"):
		return "zsh"
	case strings.HasSuffix(sh, "fish"):
		return "fish"
	case strings.HasSuffix(sh, "bash"):
		return "bash"
	default:
		return "sh"
	}
}
